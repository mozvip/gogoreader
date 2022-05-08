package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path"
	"runtime/pprof"

	"github.com/disintegration/imaging"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/mozvip/gomics/crop"
	"github.com/mozvip/gomics/files"
	"github.com/mozvip/gomics/gogoreader"
	"github.com/mozvip/gomics/resources"
	"github.com/mozvip/gomics/ui"
	"golang.org/x/image/font/basicfont"
)

type GogoReader struct {
	size         pixel.Vec
	needsRefresh bool
	infoDisplay  bool
	preferences  Preferences

	Zoom          bool
	ZoomPositionX float64
	ZoomPositionY float64

	fatalErr error

	messages []ui.Message
	win      *pixelgl.Window
}

var fontAtlas *text.Atlas

func (g *GogoReader) ToggleFullScreen() {
	if g.preferences.FullScreen {

		g.win.SetMonitor(pixelgl.PrimaryMonitor())
		g.win.SetBounds(pixel.Rect{Min: pixel.V(0, 0), Max: pixel.V(pixelgl.PrimaryMonitor().Size())})
	} else {
		g.win.SetMonitor(nil)
		g.win.SetBounds(pixel.Rect{Min: pixel.V(0, 0), Max: pixel.V(g.preferences.WindowedSize.X, g.preferences.WindowedSize.Y)})
	}
	g.size = g.win.Bounds().Max
	g.needsRefresh = true
}

func (g *GogoReader) toggleInfoDisplay() {
	g.infoDisplay = !g.infoDisplay
}

func (g *GogoReader) crop(key pixelgl.Button) float64 {
	speed := 0.0
	if g.win.Pressed(key) {
		speed = 1.0
	}
	if g.win.Repeated(key) {
		speed = 2.0
	}
	g.needsRefresh = speed > 0
	return speed
}

func (g *GogoReader) Update() error {

	if g.fatalErr != nil {
		return nil
	}

	album.GetCurrentView().Images[0].Top += g.crop(pixelgl.KeyUp)
	album.GetCurrentView().Images[0].Bottom += g.crop(pixelgl.KeyDown)
	album.GetCurrentView().Images[0].Left += g.crop(pixelgl.KeyLeft)
	album.GetCurrentView().Images[0].Right += g.crop(pixelgl.KeyRight)

	if g.win.JustPressed(pixelgl.KeyI) {
		g.toggleInfoDisplay()
	}

	if g.win.JustPressed(pixelgl.KeyF1) {
		g.win.SetSmooth(true)
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyF2) {
		g.win.SetSmooth(false)
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyPageUp) && album.CurrentViewIndex > 0 {
		g.PreviousPage()
	}

	if g.win.JustPressed(pixelgl.KeyPageDown) && album.CurrentViewIndex < len(album.Views)-1 {
		g.NextPage()
	}

	if g.win.JustPressed(pixelgl.KeyDelete) {
		// remove current page
		album.Views = append(album.Views[:album.CurrentViewIndex], album.Views[album.CurrentViewIndex+1:]...)
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyHome) {
		// go to the first page
		g.goTo(0)
	}

	if g.win.JustPressed(pixelgl.KeyEnd) {
		// go to the last page
		g.goTo(len(album.Views) - 1)
	}

	if g.win.JustPressed(pixelgl.MouseButtonLeft) {
		g.Zoom = !g.Zoom
		g.ZoomPositionX = g.win.Bounds().Center().X
	}

	if g.win.JustPressed(pixelgl.KeyL) {
		album.GetCurrentView().RotateLeft()
		g.needsRefresh = true
	}
	if g.win.JustPressed(pixelgl.KeyR) {
		album.GetCurrentView().RotateRight()
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyG) {
		album.GrayScale = !album.GrayScale
	}

	if g.win.JustPressed(pixelgl.KeyB) {
		if g.win.Pressed(pixelgl.KeyLeftShift) || g.win.Pressed(pixelgl.KeyRightShift) {
			// only for the current page
			album.GetCurrentView().ToggleBorder(g.preferences.RemoveBorders)
		} else {
			g.preferences.RemoveBorders = !g.preferences.RemoveBorders
		}
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyBackspace) {
		album.Reset()
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyMinus) {
		album.GetCurrentView().RotationAngle -= 0.05
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyPeriod) {
		album.GetCurrentView().RotationAngle += 0.05
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyKPDivide) {
		album.GetCurrentView().RotationAngle = 0
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyF11) || g.win.JustPressed(pixelgl.KeyF) {
		if !g.preferences.FullScreen {
			// save the current size of the window
			g.preferences.WindowedSize = g.win.Bounds().Size()
		}
		g.preferences.FullScreen = !g.preferences.FullScreen
		g.ToggleFullScreen()
	}

	if g.win.JustPressed(pixelgl.KeyEscape) || g.win.JustPressed(pixelgl.KeyQ) {
		AppQuit(g.preferences)
	}

	if g.win.JustPressed(pixelgl.KeyD) {
		if len(album.GetCurrentView().Images) == 1 && album.CurrentViewIndex < len(album.Views)-1 {
			// only if we have a page after the current one
			album.GetCurrentView().Images = append(album.GetCurrentView().Images, album.Views[album.CurrentViewIndex+1].Images...)
			album.Views = append(album.Views[:album.CurrentViewIndex+1], album.Views[album.CurrentViewIndex+2:]...)
		} else if len(album.GetCurrentView().Images) > 1 {
			// create a new page with only the second image
			newPage := ViewData{Images: album.GetCurrentView().Images[1:]}
			// only keep the first image on the current page
			album.GetCurrentView().Images = album.GetCurrentView().Images[:1]

			// allocate one more page
			album.Views = append(album.Views[:album.CurrentViewIndex+1], album.Views[album.CurrentViewIndex:]...)
			// next page is the new page
			album.Views[album.CurrentViewIndex+1] = &newPage
		}
		g.needsRefresh = true
	}

	return g.refresh()
}

func (g *GogoReader) drawBackGround() {
	imd := imdraw.New(nil)
	if album.GetCurrentView().BackgroundColors != nil {
		backw := g.size.X / float64(len(album.GetCurrentView().BackgroundColors))
		x := 0.0
		for _, color := range album.GetCurrentView().BackgroundColors {
			imd.Color = color
			imd.Push(pixel.V(x, 0.0), pixel.V(x+backw, 0.0))
			imd.Push(pixel.V(x+backw, g.size.Y), pixel.V(x, g.size.Y))
			imd.Polygon(0)
			x += backw
		}
		imd.Draw(g.win)
	}
}

func (g *GogoReader) Draw() {

	// draw background
	g.drawBackGround()

	pageData := album.GetCurrentView()

	var totalWidth, maxHeight float64
	for _, sprite := range pageData.imageSprites {
		spriteW, spriteH := sprite.Frame().W(), sprite.Frame().H()
		totalWidth += spriteW
		if spriteH > maxHeight {
			maxHeight = spriteH
		}
	}

	// draw scaled images
	scale := 1.0
	if maxHeight > g.size.Y {
		scale = g.size.Y / maxHeight
	}
	/*
		FIXME
			if (totalWidth * scale) > g.size.X {
				scale = g.size.X / totalWidth
			}
	*/

	if g.Zoom {
		mousePosition := g.win.MousePosition()

		// FIXME : use a percentage of the total height instead of absolute 100
		if mousePosition.Y < 100 {
			if g.ZoomPositionY >= g.size.Y-maxHeight {
				g.ZoomPositionY -= 15
			}
		}
		if mousePosition.Y > (g.size.Y - 100) {
			if g.ZoomPositionY <= maxHeight {
				g.ZoomPositionY += 15
			}
		}
	}

	center := g.win.Bounds().Center()
	positions := make([]pixel.Vec, 0, len(pageData.imageSprites))
	startX := center.X - (totalWidth / 2.0)
	for _, sprite := range pageData.imageSprites {
		var imageW = sprite.Frame().W()
		positions = append(positions, pixel.Vec{X: startX + imageW/2.0, Y: center.Y})
		startX += sprite.Frame().W()
	}
	for index, sprite := range pageData.imageSprites {
		matrix := pixel.IM.Moved(positions[index])
		if g.Zoom {
			scale = g.win.Bounds().W() / totalWidth
			matrix = matrix.Scaled(pixel.V(g.ZoomPositionX, g.ZoomPositionY), scale)
		} else {
			matrix = matrix.Scaled(g.win.Bounds().Center(), scale)
		}
		sprite.Draw(g.win, matrix)
	}

	if g.infoDisplay {

		textScale := 2.0
		infoText := text.New(pixel.V(5, g.size.Y-fontAtlas.LineHeight()*textScale), fontAtlas)
		var fileNames string
		for i, image := range album.GetCurrentView().Images {
			if i > 0 {
				fileNames = fileNames + " "
			}
			fileNames = fileNames + image.FileName
		}

		message := fmt.Sprintf("Page %d (%d %%)\nFiles names\t%s\nScreen Size\t%.0f x %.0f\nImage Size\t%.0f x %.0f\nscale %.2f", album.CurrentViewIndex, album.CurrentViewIndex*100/len(album.Views), fileNames, g.size.X, g.size.Y, totalWidth, maxHeight, scale)
		fmt.Fprintln(infoText, message)
		// fmt.Fprintf(infoText, "Rotation : x=%.0f y=%.0f", g.ZoomPositionX, g.ZoomPositionY)
		if g.Zoom {
			fmt.Fprintf(infoText, "Zoom position : x=%.0f y=%.0f", g.ZoomPositionX, g.ZoomPositionY)
		}

		infoBoxW := infoText.Bounds().Max.X
		infoBoxH := infoText.Bounds().Max.Y

		imd := imdraw.New(nil)
		imd.Color = pixel.RGBA{0.2, 0.2, 0.2, 0.5}
		imd.Push(pixel.V(0, g.size.Y))
		imd.Push(pixel.V(infoBoxW, g.size.Y-infoBoxH))
		imd.Rectangle(0)

		imd.Draw(g.win)

		infoText.Draw(g.win, pixel.IM.Scaled(infoText.Orig, textScale))
	}

	y := 0
	for i := 0; i < len(g.messages); i++ {
		//g.messages[i].Draw(screen, fontFace, 0, y)
		y += 40
	}

}

var comicBook files.ComicBookArchive
var configFolder string
var archiveFile string

var album Album

func (g *GogoReader) NextPage() bool {
	g.goTo(album.CurrentViewIndex + 1)
	return true
}

func (g *GogoReader) PreviousPage() bool {
	g.goTo(album.CurrentViewIndex - 1)
	return true
}

func AppQuit(preferences Preferences) {
	saveConfiguration(preferences)
	if *cpuprofile != "" {
		pprof.StopCPUProfile()
	}
	os.Exit(0)
}

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func backgroundColor(pictureData *pixel.PictureData, rect pixel.Rect) pixel.RGBA {
	return gogoreader.ProminentColor(pictureData, rect)
}

func (g *GogoReader) prepareView(viewData *ViewData) error {

	viewData.mu.Lock()
	defer viewData.mu.Unlock()

	if viewData.imageSprites != nil {
		// page was already prepared
		return nil
	}

	var err error
	var totalWidth, h float64

	viewData.BackgroundColors = make([]pixel.RGBA, 0, 2)
	viewData.imageSprites = make([]*pixel.Sprite, 0, len(viewData.Images))
	for index, imgData := range viewData.Images {
		// ensure all images used by this page are loaded
		var rawImage image.Image
		rawImage, err = comicBook.ReadEntry(imgData.FileName)
		if err != nil {
			log.Printf("Error reading image %s - %s\n", imgData.FileName, err.Error())
			return err
		}
		if imgData.Rotation != None {
			if imgData.Rotation == Left {
				rawImage = imaging.Rotate90(rawImage)
			} else if imgData.Rotation == Right {
				rawImage = imaging.Rotate270(rawImage)
			}
		}
		if album.GrayScale {
			rawImage = imaging.Grayscale(rawImage)
		}
		if viewData.RotationAngle != 0 {
			rawImage = imaging.Rotate(rawImage, viewData.RotationAngle, color.RGBA{255, 255, 255, 255})
		}

		pictureData := pixel.PictureDataFromImage(rawImage)

		cropRect := pictureData.Bounds()
		if imgData.Left > 0 || imgData.Right > 0 || imgData.Bottom > 0 || imgData.Top > 0 {
			cropRect = pixel.Rect{Min: pixel.V(cropRect.Min.X+imgData.Left, cropRect.Min.Y+imgData.Bottom), Max: pixel.V(cropRect.Max.X-imgData.Right, cropRect.Max.Y-imgData.Top)}
		}

		if (viewData.bordersOverride && viewData.RemoveBorders) || g.preferences.RemoveBorders {
			crop.CropBorders(pictureData, &cropRect)
		}

		w := cropRect.W() / 5

		offsetW := cropRect.W() / 20

		if index == 0 {
			rect := pixel.Rect{Min: pixel.V(cropRect.Min.X+offsetW, cropRect.Min.Y), Max: pixel.V(cropRect.Min.X+w, cropRect.Max.Y)}
			viewData.BackgroundColors = append(viewData.BackgroundColors, backgroundColor(pictureData, rect))
		}
		if index == len(viewData.Images)-1 {
			rect := pixel.Rect{Min: pixel.V(cropRect.Max.X-w, cropRect.Min.Y), Max: pixel.V(cropRect.Max.X-offsetW, cropRect.Max.Y)}
			viewData.BackgroundColors = append(viewData.BackgroundColors, backgroundColor(pictureData, rect))
		}

		iw, ih := float64(cropRect.W()), float64(cropRect.H())
		totalWidth += iw
		if ih > h {
			h = ih
		}

		sprite := pixel.NewSprite(pictureData, cropRect)
		viewData.imageSprites = append(viewData.imageSprites, sprite)
	}

	return err
}

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	configFolder = path.Join(userConfigDir, "gogoreader")
	err = os.MkdirAll(configFolder, 0600)
	if err != nil {
		panic(err)
	}

	var logFile *os.File
	logFileName := path.Join(configFolder, "gogoreader.log")
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Unable to open log file %s for writing, will log to console instead\n", logFileName)
	} else {
		// setup logging to file
		log.Printf("Log directed to file :  %s\n", logFileName)
		log.SetOutput(logFile)
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func run() {

	var err error

	archiveFile = flag.Args()[0]
	log.Println("Opening file ", archiveFile)

	var icons []pixel.Picture
	image, _, _ := image.Decode(bytes.NewReader(resources.Gogoreader_png))
	icons = append(icons, pixel.PictureDataFromImage(image))

	g := &GogoReader{}

	log.Printf("Loading %s\n", archiveFile)
	comicBook, err = files.FromFile(archiveFile)
	if err != nil {
		g.fatalErr = err
	} else {
		defer comicBook.Close()
		err = comicBook.Init()
		if err != nil {
			g.fatalErr = err
		}
	}

	if g.fatalErr != nil {
		g.preferences.FullScreen = false
		g.preferences.WindowedSize.X = 500
		g.preferences.WindowedSize.Y = 100
	} else {
		g.preferences, err = readConfiguration(comicBook.GetMD5())
		if err != nil {
			g.fatalErr = err
		}
	}

	g.needsRefresh = true

	var monitor *pixelgl.Monitor
	if g.preferences.FullScreen {
		monitor = pixelgl.PrimaryMonitor()
	}

	cfg := pixelgl.WindowConfig{
		Title:     archiveFile,
		Bounds:    pixel.R(0, 0, g.preferences.WindowedSize.X, g.preferences.WindowedSize.Y),
		Monitor:   monitor,
		Resizable: g.fatalErr == nil,
		Icon:      icons,
		VSync:     true,
	}
	g.win, err = pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	fontAtlas = text.NewAtlas(basicfont.Face7x13, text.ASCII)

	if g.fatalErr != nil {
		g.win.SetTitle(fmt.Sprintf("Error - Unable to display %s", archiveFile))
		textScale := 1.0
		errorText := text.New(pixel.V(10, 0+fontAtlas.LineHeight()*textScale), fontAtlas)
		fmt.Fprintf(errorText, "%s\n%s", archiveFile, g.fatalErr.Error())
		g.win.SetBounds(errorText.Bounds())
		for !g.win.Closed() {
			errorText.Draw(g.win, pixel.IM.Scaled(errorText.Orig, textScale))
			g.win.Update()
		}

	} else {
		g.win.SetSmooth(true)
		g.ToggleFullScreen()
		g.refresh()

		for !g.win.Closed() {
			g.Update()
			g.Draw()
			g.win.Update()
		}
	}

	if g.fatalErr != nil {
		os.Exit(-1)
	} else {
		AppQuit(g.preferences)
	}
}

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Missing command line parameter : file name")
	}

	flag.Parse()

	log.Println(flag.Args())

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	pixelgl.Run(run)
}

func (g *GogoReader) goTo(newImageIndex int) error {
	if newImageIndex == album.CurrentViewIndex {
		return nil
	}
	album.CurrentViewIndex = newImageIndex
	g.needsRefresh = true
	return nil
}

func (g *GogoReader) refresh() error {

	if !g.needsRefresh {
		return nil
	}
	g.needsRefresh = false
	album.GetCurrentView().imageSprites = nil
	err := g.prepareView(album.GetCurrentView())
	if err != nil {
		return err
	}
	if album.CurrentViewIndex < len(album.Views)-1 {
		// prepare next page in the background
		go g.prepareView(album.Views[album.CurrentViewIndex+1])
	}

	return err
}
