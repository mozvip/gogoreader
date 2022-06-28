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
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/mozvip/gomics/crop"
	"github.com/mozvip/gomics/files"
	"github.com/mozvip/gomics/gogoreader"
	"github.com/mozvip/gomics/resources"
	"github.com/mozvip/gomics/ui"
)

type GogoReader struct {
	needsRefresh bool
	infoDisplay  bool
	preferences  Preferences

	Zoom          bool
	ZoomPositionX uint32
	ZoomPositionY uint32

	fatalErr error

	messages []ui.Message
}

func (g *GogoReader) ToggleFullScreen() {
	if g.preferences.FullScreen {
		if !rl.IsWindowFullscreen() {
			monitor := rl.GetCurrentMonitor()
			width := int32(rl.GetMonitorWidth(monitor))
			height := int32(rl.GetMonitorHeight(monitor))
			rl.CloseWindow()
			rl.InitWindow(width, height, "GogoReader")
			rl.ToggleFullscreen()
		}
	} else {
		if rl.IsWindowFullscreen() {
			rl.CloseWindow()
			rl.InitWindow(g.preferences.WindowedSize.X, g.preferences.WindowedSize.Y, "GogoReader")
		}
	}
	g.needsRefresh = true
}

func (g *GogoReader) toggleInfoDisplay() {
	g.infoDisplay = !g.infoDisplay
}

func (g *GogoReader) crop(key int32) float32 {
	speed := float32(0.0)
	if rl.IsKeyPressed(key) {
		speed = 1.0
	}
	// FIXME
	/*
		if g.win.Repeated(key) {
			speed = 2.0
		}
	*/
	g.needsRefresh = speed > 0
	return speed
}

func (g *GogoReader) Update() error {

	album.GetCurrentView().Images[0].Top += g.crop(rl.KeyUp)
	album.GetCurrentView().Images[0].Bottom += g.crop(rl.KeyDown)
	album.GetCurrentView().Images[0].Left += g.crop(rl.KeyLeft)
	album.GetCurrentView().Images[0].Right += g.crop(rl.KeyRight)

	if rl.IsKeyPressed(rl.KeyI) {
		g.toggleInfoDisplay()
	}

	if rl.IsKeyPressed(rl.KeyF1) {
		//TODO g.win.SetSmooth(true)
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyF2) {
		//TODO g.win.SetSmooth(false)
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyPageUp) && album.CurrentViewIndex > 0 {
		g.PreviousPage()
	}

	if rl.IsKeyPressed(rl.KeyPageDown) && album.CurrentViewIndex < len(album.Views)-1 {
		g.NextPage()
	}

	if rl.IsKeyPressed(rl.KeyDelete) {
		// remove current page
		album.Views = append(album.Views[:album.CurrentViewIndex], album.Views[album.CurrentViewIndex+1:]...)
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyHome) {
		// go to the first page
		g.goTo(0)
	}

	if rl.IsKeyPressed(rl.KeyEnd) {
		// go to the last page
		g.goTo(len(album.Views) - 1)
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		g.Zoom = !g.Zoom
		//FIXME g.ZoomPositionX = g.win.Bounds().Center().X
	}

	if rl.IsKeyPressed(rl.KeyL) {
		album.GetCurrentView().RotateLeft()
		g.needsRefresh = true
	}
	if rl.IsKeyPressed(rl.KeyR) {
		album.GetCurrentView().RotateRight()
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyG) {
		album.GrayScale = !album.GrayScale
	}

	if rl.IsKeyPressed(rl.KeyB) {
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			// only for the current page
			album.GetCurrentView().ToggleBorder(g.preferences.RemoveBorders)
		} else {
			g.preferences.RemoveBorders = !g.preferences.RemoveBorders
		}
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyBackspace) {
		album.Reset()
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyMinus) {
		album.GetCurrentView().RotationAngle -= 0.05
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyPeriod) {
		album.GetCurrentView().RotationAngle += 0.05
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyKpDivide) {
		album.GetCurrentView().RotationAngle = 0
		g.needsRefresh = true
	}

	if rl.IsKeyPressed(rl.KeyF11) || rl.IsKeyPressed(rl.KeyF) {
		if !g.preferences.FullScreen {
			// save the current size of the window
			g.preferences.WindowedSize = NewVector2Int(rl.GetScreenWidth(), rl.GetScreenHeight())
		}
		g.preferences.FullScreen = !g.preferences.FullScreen
		g.ToggleFullScreen()
	}

	if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyQ) {
		AppQuit(g.preferences)
	}

	if rl.IsKeyPressed(rl.KeyD) {
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
	if album.GetCurrentView().BackgroundColors != nil {
		var x, backw int32
		backw = int32(rl.GetScreenWidth()) / int32(len(album.GetCurrentView().BackgroundColors))
		x = 0
		for _, color := range album.GetCurrentView().BackgroundColors {
			rl.DrawRectangle(x, 0, x+backw, int32(rl.GetScreenHeight()), color)
			x += backw
		}
	}
}

func (g *GogoReader) Draw() {

	// draw background
	g.drawBackGround()

	currentView := album.GetCurrentView()

	totalWidth := currentView.TotalWidth
	maxHeight := currentView.MaxHeight

	// scale image to maximize visibility
	scale := float32(rl.GetScreenHeight()) / float32(maxHeight)

	if g.Zoom {
		mousePosition := rl.GetMousePosition()

		// FIXME : use a percentage of the total height instead of absolute 100
		if mousePosition.Y < 100 {
			if g.ZoomPositionY >= uint32(rl.GetScreenHeight())-maxHeight {
				g.ZoomPositionY -= 15
			}
		}
		if mousePosition.Y > float32(rl.GetScreenHeight()-100) {
			if g.ZoomPositionY <= maxHeight {
				g.ZoomPositionY += 15
			}
		}
	}

	positions := make([]rl.Vector2, 0, len(currentView.images))
	wPerImage := rl.GetScreenWidth() / len(currentView.images)
	startX := wPerImage / 4
	for i := 0; i < len(currentView.images); i++ {
		positions = append(positions, rl.NewVector2(float32(startX), 0))
		startX += wPerImage
	}
	if currentView.textures == nil {
		currentView.textures = make([]rl.Texture2D, len(currentView.images))
		for index, image := range currentView.images {
			currentView.textures[index] = rl.LoadTextureFromImage(image)
		}
	}
	for index, texture := range currentView.textures {
		rl.DrawTextureEx(texture, rl.NewVector2(positions[index].X, positions[index].Y), 0.0, scale, rl.White)
	}

	if g.infoDisplay {

		var fileNames string
		for i, image := range album.GetCurrentView().Images {
			if i > 0 {
				fileNames = fileNames + " "
			}
			fileNames = fileNames + image.FileName
		}

		message := fmt.Sprintf("Page %d (%d %%)\nFiles names\t%s\nScreen Size\t%d x %d\nImage Size\t%d x %d\nscale %.2f", album.CurrentViewIndex, album.CurrentViewIndex*100/len(album.Views), fileNames, rl.GetScreenWidth(), rl.GetScreenHeight(), totalWidth, maxHeight, scale)
		if g.Zoom {
			message = message + fmt.Sprintf("\nZoom position : x=%d y=%d", g.ZoomPositionX, g.ZoomPositionY)
		}
		// fmt.Fprintf(infoText, "Rotation : x=%.0f y=%.0f", g.ZoomPositionX, g.ZoomPositionY)
		textWidth := rl.MeasureText(message, 30)
		rl.DrawRectangle(0, 0, textWidth+10, 200, color.RGBA{40, 40, 40, 128})
		rl.DrawText(message, 5, 0, 30, rl.White)
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

func backgroundColor(pictureData *image.NRGBA, rect rl.Rectangle) color.RGBA {
	return gogoreader.ProminentColor(pictureData, rect)
}

func (g *GogoReader) prepareView(viewData *ViewData) error {

	viewData.mu.Lock()
	defer viewData.mu.Unlock()

	if viewData.images != nil {
		// page was already prepared
		return nil
	}

	var err error
	var totalWidth, maxHeight uint32

	viewData.BackgroundColors = make([]color.RGBA, 0, 2)
	viewData.images = make([]*rl.Image, 0, len(viewData.Images))
	for index, imgData := range viewData.Images {
		// ensure all images used by this view are loaded
		var rawImage *image.NRGBA
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
		cropRect := rl.NewRectangle(0, 0, float32(rawImage.Bounds().Max.X), float32(rawImage.Bounds().Max.Y))
		if imgData.Left > 0 || imgData.Right > 0 || imgData.Bottom > 0 || imgData.Top > 0 {
			cropRect = rl.NewRectangle(imgData.Left, imgData.Top, cropRect.Width-(imgData.Left+imgData.Right), cropRect.Height-(imgData.Top+imgData.Bottom))
		}

		if (viewData.bordersOverride && viewData.RemoveBorders) || g.preferences.RemoveBorders {
			crop.CropBorders(rawImage, &cropRect)
		}

		blockWidth := cropRect.Width / 5
		if index == 0 {
			rect := rl.NewRectangle(cropRect.X, cropRect.Y, blockWidth, cropRect.Height)
			viewData.BackgroundColors = append(viewData.BackgroundColors, backgroundColor(rawImage, rect))
		}
		if index == len(viewData.Images)-1 {
			rect := rl.NewRectangle(cropRect.X+cropRect.Width-blockWidth, cropRect.Y, blockWidth, cropRect.Height)
			viewData.BackgroundColors = append(viewData.BackgroundColors, backgroundColor(rawImage, rect))
		}

		iw, ih := uint32(cropRect.Width), uint32(cropRect.Height)
		totalWidth += iw
		if ih > maxHeight {
			maxHeight = ih
		}

		pictureData := rl.NewImageFromImage(rawImage)
		rl.ImageCrop(pictureData, cropRect)
		viewData.images = append(viewData.images, pictureData)
	}

	viewData.TotalWidth = totalWidth
	viewData.MaxHeight = maxHeight

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

	rl.InitWindow(g.preferences.WindowedSize.X, g.preferences.WindowedSize.Y, archiveFile)
	image, _, _ := image.Decode(bytes.NewReader(resources.Gogoreader_png))
	rl.SetWindowIcon(*rl.NewImageFromImage(image))

	if g.fatalErr != nil {
		rl.SetWindowTitle(fmt.Sprintf("Error - Unable to display %s", archiveFile))
		fontSize := int32(25)
		errorText := g.fatalErr.Error()
		width := rl.MeasureText(errorText, fontSize)
		rl.SetWindowSize(int(width), 80)
		for !rl.WindowShouldClose() {
			rl.BeginDrawing()
			rl.ClearBackground(rl.Black)
			rl.DrawText(errorText, 0, 0, fontSize, rl.White)
			rl.EndDrawing()
		}

	} else {
		// FIXME g.win.SetSmooth(true)
		g.ToggleFullScreen()
		g.refresh()

		for !rl.WindowShouldClose() {
			rl.BeginDrawing()

			g.Update()
			g.Draw()

			rl.EndDrawing()
		}
	}

	rl.CloseWindow()

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

	run()
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
	view := album.GetCurrentView()
	if view.images != nil {
		for _, image := range view.images {
			rl.UnloadImage(image)
		}
		view.images = nil
	}
	err := g.prepareView(view)
	if err != nil {
		return err
	}
	if album.CurrentViewIndex < len(album.Views)-1 {
		// prepare next page in the background
		go g.prepareView(album.Views[album.CurrentViewIndex+1])
	}

	return err
}
