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
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

type Gomics struct {
	size         pixel.Vec
	needsRefresh bool
	infoDisplay  bool
	preferences  Preferences
	Zoom         bool

	fatalErr error

	imageSprites []*pixel.Sprite
	messages     []ui.Message
	win          *pixelgl.Window
}

var logFile *os.File
var fontFace font.Face

func (g *Gomics) InitFullScreen() {
	if g.preferences.FullScreen {
		g.win.SetMonitor(pixelgl.PrimaryMonitor())
	} else {
		g.win.SetMonitor(nil)
	}
	g.size = g.win.Bounds().Max
}

func (g *Gomics) toggleInfoDisplay() {
	g.infoDisplay = !g.infoDisplay
}

func (g *Gomics) crop(key pixelgl.Button) int {
	speed := 0
	if g.win.Pressed(key) {
		speed = 1
	} else if g.win.Repeated(key) {
		speed = 2
	}
	g.needsRefresh = speed > 0
	return speed
}

func (g *Gomics) Update() error {

	if g.fatalErr != nil {
		return nil
	}

	album.GetCurrentPage().Images[0].Top += g.crop(pixelgl.KeyUp)
	album.GetCurrentPage().Images[0].Bottom += g.crop(pixelgl.KeyDown)
	album.GetCurrentPage().Images[0].Left += g.crop(pixelgl.KeyLeft)
	album.GetCurrentPage().Images[0].Right += g.crop(pixelgl.KeyRight)

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

	if g.win.JustPressed(pixelgl.KeyPageUp) && album.CurrentPageIndex > 0 {
		g.PreviousPage()
	}

	if g.win.JustPressed(pixelgl.KeyPageDown) && album.CurrentPageIndex < len(album.Pages)-1 {
		g.NextPage()
	}

	if g.win.JustPressed(pixelgl.KeyDelete) {
		// remove current page
		album.Pages = append(album.Pages[:album.CurrentPageIndex], album.Pages[album.CurrentPageIndex+1:]...)
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyHome) {
		// go to the first page
		g.goTo(0)
	}

	if g.win.JustPressed(pixelgl.KeyEnd) {
		// go to the last page
		g.goTo(len(album.Pages) - 1)
	}

	if g.win.JustPressed(pixelgl.MouseButtonLeft) {
		g.Zoom = !g.Zoom
	}

	if g.win.JustPressed(pixelgl.KeyL) {
		album.GetCurrentPage().RotateLeft()
		g.needsRefresh = true
	}
	if g.win.JustPressed(pixelgl.KeyR) {
		album.GetCurrentPage().RotateRight()
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyG) {
		album.GrayScale = !album.GrayScale
	}

	if g.win.JustPressed(pixelgl.KeyB) {
		g.preferences.RemoveBorders = !g.preferences.RemoveBorders
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyBackspace) {
		album.Reset()
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyKPSubtract) {
		album.GetCurrentPage().RotationAngle -= 0.05
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyKPAdd) {
		album.GetCurrentPage().RotationAngle += 0.05
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyKPDivide) {
		album.GetCurrentPage().RotationAngle = 0
		g.needsRefresh = true
	}

	if g.win.JustPressed(pixelgl.KeyF11) || g.win.JustPressed(pixelgl.KeyF) {

		if !g.preferences.FullScreen {
			// save the size of the window
			g.preferences.WindowedSize = g.win.Bounds().Size()
		}
		g.preferences.FullScreen = !g.preferences.FullScreen
		g.InitFullScreen()
	}

	if g.win.JustPressed(pixelgl.KeyEscape) || g.win.JustPressed(pixelgl.KeyQ) {
		AppQuit(g.preferences)
	}

	if g.win.JustPressed(pixelgl.KeyLeftShift) {
		if len(album.GetCurrentPage().Images) == 1 && album.CurrentPageIndex < len(album.Pages)-1 {
			// only if we have a page after the current one
			album.GetCurrentPage().Images = append(album.GetCurrentPage().Images, album.Pages[album.CurrentPageIndex+1].Images...)
			album.Pages = append(album.Pages[:album.CurrentPageIndex+1], album.Pages[album.CurrentPageIndex+2:]...)
		} else if len(album.GetCurrentPage().Images) > 1 {
			//newPage := PageData{Images: album.GetCurrentPage().Images[1:]}
			album.GetCurrentPage().Images = album.GetCurrentPage().Images[:1]
			//album.Pages = append(album.Pages[:album.CurrentPageIndex], newPage, album.Pages[album.CurrentPageIndex+2:]...)
		}
		g.needsRefresh = true
	}

	return g.refresh()
}

func (g *Gomics) Draw() {

	g.win.Clear(colornames.Black)

	if g.fatalErr != nil {
		basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
		basicTxt := text.New(pixel.V(0, 45), basicAtlas)
		fmt.Fprintln(basicTxt, g.fatalErr.Error())
		return
	}

	imd := imdraw.New(nil)

	if album.GetCurrentPage().BackgroundColors != nil {
		backw := g.size.X / float64(len(album.GetCurrentPage().BackgroundColors))
		x := 0.0
		for _, color := range album.GetCurrentPage().BackgroundColors {
			imd.Color = color
			imd.Push(pixel.V(x, 0.0), pixel.V(x+backw, 0.0))
			imd.Push(pixel.V(x+backw, g.size.Y), pixel.V(x, g.size.Y))
			imd.Polygon(0)
			x += backw
		}
		imd.Draw(g.win)
	}

	var totalWidth, maxHeight float64
	for _, sprite := range g.imageSprites {
		spriteW, spriteH := sprite.Picture().Bounds().W(), sprite.Picture().Bounds().H()
		totalWidth += spriteW
		if spriteH > maxHeight {
			maxHeight = spriteH
		}
	}

	scale := 1.0
	if maxHeight > g.size.Y {
		scale = g.size.Y / maxHeight
	}
	if (totalWidth * scale) > g.size.X {
		scale = g.size.X / totalWidth
	}

	tx := (g.size.X - totalWidth*scale) / 2.0
	ty := (g.size.Y - maxHeight*scale) / 2.0

	center := g.win.Bounds().Center()
	positions := make([]pixel.Vec, 0, len(g.imageSprites))
	startX := center.X - (totalWidth / 2.0)
	for _, sprite := range g.imageSprites {
		var imageW = sprite.Picture().Bounds().W()
		positions = append(positions, pixel.Vec{X: startX + imageW/2.0, Y: center.Y})
		startX += sprite.Picture().Bounds().W()
	}
	for index, sprite := range g.imageSprites {
		matrix := pixel.IM.Moved(positions[index])
		matrix = matrix.Scaled(g.win.Bounds().Center(), scale)
		sprite.Draw(g.win, matrix)
	}

	if g.infoDisplay {
		// message := fmt.Sprintf("%0.2f TPS\n%d %%\nscale %.2f\nangle %f", ebiten.CurrentTPS(), album.CurrentPageIndex*100/len(album.Pages), album.GetCurrentPage().scale, album.GetCurrentPage().RotationAngle)
		message := fmt.Sprintf("Page %d (%d %%)\nScreen Size %d x %d\nImage Size %d x %d\ntx=%.2f ty=%.2f\nscale %.2f", album.CurrentPageIndex, album.CurrentPageIndex*100/len(album.Pages), g.size.X, g.size.Y, totalWidth, maxHeight, tx, ty, scale)
		log.Println(message)
		//ebitenutil.DebugPrint(screen, message)
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

type Size struct {
	w, h float64
}

var album Album

func (g *Gomics) NextPage() bool {
	g.goTo(album.CurrentPageIndex + 1)
	return true
}

func (g *Gomics) PreviousPage() bool {
	g.goTo(album.CurrentPageIndex - 1)
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

func backgroundColor(image image.Image, rect image.Rectangle) pixel.RGBA {
	subImager, ok := image.(SubImager)
	if ok {
		r, g, b := gogoreader.ProminentColor(subImager.SubImage(rect))
		return pixel.RGBA{R: float64(r) / 255.0, G: float64(g) / 255.0, B: float64(b) / 255.0, A: 1.0}
	} else {
		log.Println("Image does not implement SubImage, unable to compute background colors")
	}
	return pixel.RGBA{R: 0, G: 0, B: 0, A: 1.0}
}

func (g *Gomics) preparePage() error {

	if g.imageSprites != nil {
		// image was already prepared
		return nil
	}

	var err error
	var totalWidth, h float64

	pageData := album.GetCurrentPage()
	pageData.BackgroundColors = make([]pixel.RGBA, 0, 2)
	g.imageSprites = make([]*pixel.Sprite, 0, len(pageData.Images))
	for index, imgData := range pageData.Images {
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
		if pageData.RotationAngle != 0 {
			rawImage = imaging.Rotate(rawImage, pageData.RotationAngle, color.RGBA{255, 255, 255, 255})
		}

		cropRect := rawImage.Bounds()
		if imgData.Left > 0 || imgData.Right > 0 || imgData.Bottom > 0 || imgData.Top > 0 {
			cropRect = image.Rect(cropRect.Min.X+imgData.Left, cropRect.Min.Y+imgData.Top, cropRect.Max.X-imgData.Right, cropRect.Max.Y-imgData.Bottom)
		}

		if g.preferences.RemoveBorders {
			crop.CropBorders(rawImage, &cropRect)
			if cropRect.Dx() > 1 && cropRect.Dy() > 1 {
				subImager, ok := rawImage.(SubImager)
				if ok {
					rawImage = subImager.SubImage(cropRect)
				} else {
					log.Println("Image does not implement SubImage, unable to remove borders")
				}
			}
		}

		bounds := rawImage.Bounds()
		w := bounds.Dx() / 4
		if index == 0 {
			rect := image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: w, Y: bounds.Max.Y}}
			pageData.BackgroundColors = append(pageData.BackgroundColors, backgroundColor(rawImage, rect))
		}
		if index == len(pageData.Images)-1 {
			rect := image.Rectangle{Min: image.Point{X: 3 * w, Y: 0}, Max: image.Point{X: bounds.Max.X, Y: bounds.Max.Y}}
			pageData.BackgroundColors = append(pageData.BackgroundColors, backgroundColor(rawImage, rect))
		}

		pictureData := pixel.PictureDataFromImage(rawImage)
		bounds = pictureData.Image().Bounds()

		iw, ih := float64(bounds.Dx()), float64(bounds.Dy())
		totalWidth += iw
		if ih > h {
			h = ih
		}

		sprite := pixel.NewSprite(pictureData, pictureData.Bounds())
		g.imageSprites = append(g.imageSprites, sprite)
	}

	return err
}

func init() {
	var err error

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	configFolder = path.Join(userConfigDir, "gogoreader")
	err = os.MkdirAll(configFolder, 0600)
	if err != nil {
		panic(err)
	}

	logFile, err = os.OpenFile(path.Join(configFolder, "gogoreader.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)

	tt, err := opentype.Parse(resources.Pacifico_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	fontFace, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func run() {

	var err error

	if len(os.Args) < 2 {
		log.Fatal("Need param")
	}

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	archiveFile = flag.Args()[0]
	log.Println("Opening file ", archiveFile)

	var icons []pixel.Picture
	image, _, _ := image.Decode(bytes.NewReader(resources.Gogoreader_png))
	icons = append(icons, pixel.PictureDataFromImage(image))

	g := &Gomics{}

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
		Resizable: true,
		Icon:      icons,
		VSync:     true,
	}
	g.win, err = pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	g.win.SetSmooth(true)
	g.InitFullScreen()
	g.preparePage()

	for !g.win.Closed() {
		g.Update()
		g.Draw()
		g.win.Update()
	}

	if g.fatalErr != nil {
		os.Exit(-1)
	} else {
		AppQuit(g.preferences)
	}
}

func main() {
	pixelgl.Run(run)
}

func (g *Gomics) goTo(newImageIndex int) error {
	if newImageIndex == album.CurrentPageIndex {
		return nil
	}
	album.CurrentPageIndex = newImageIndex
	g.needsRefresh = true
	return nil
}

func (g *Gomics) refresh() error {

	if !g.needsRefresh {
		return nil
	}
	g.needsRefresh = false
	g.imageSprites = nil

	err := g.preparePage()
	if err != nil {
		return err
	}
	/*
		if album.CurrentPageIndex < len(album.Pages)-1 {
			// prepare next page in the background
			go g.preparePage(&album.Pages[album.CurrentPageIndex+1])
		}
	*/

	return err
}
