package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mozvip/gomics/crop"
	"github.com/mozvip/gomics/files"
	"github.com/mozvip/gomics/resources"
	"github.com/mozvip/gomics/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type Gomics struct {
	currentImage    *ebiten.Image
	prominentColors []color.RGBA
	size            Size
	needsRefresh    bool
	infoDisplay     bool
	preferences     Preferences
	Zoom            bool

	fatalErr error

	messages []ui.Message
}

var logFile *os.File
var fontFace font.Face

func (g *Gomics) InitFullScreen() {
	ebiten.SetFullscreen(g.preferences.FullScreen)
	if !g.preferences.FullScreen {
		// restore the size of the window
		g.size = g.preferences.WindowedSize
	} else {
		g.size.w, g.size.h = ebiten.ScreenSizeInFullscreen()
	}
}

func (g *Gomics) toggleInfoDisplay() {
	g.infoDisplay = !g.infoDisplay
}

func (g *Gomics) Update() error {

	if g.fatalErr != nil {
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		g.toggleInfoDisplay()
	}

	if inpututil.KeyPressDuration(ebiten.KeyUp) > 0 {
		speed := 1
		if inpututil.KeyPressDuration(ebiten.KeyUp) > 200 {
			speed = 3
		} else if inpututil.KeyPressDuration(ebiten.KeyUp) > 100 {
			speed = 2
		}
		album.Pages[album.CurrentPage].Top += speed
		g.needsRefresh = true
	}

	if inpututil.KeyPressDuration(ebiten.KeyDown) > 0 {
		speed := 1
		if inpututil.KeyPressDuration(ebiten.KeyUp) > 200 {
			speed = 3
		} else if inpututil.KeyPressDuration(ebiten.KeyUp) > 100 {
			speed = 2
		}
		album.Pages[album.CurrentPage].Bottom += speed
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.NextPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.preferences.Filter = LANCZOS
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.preferences.Filter = NEAREST_NEIGHBOR
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.PreviousPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		album.Pages[album.CurrentPage].Visible = false
		if !g.NextPage() {
			g.PreviousPage()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		// go to the first visible page
		for i := 0; i < len(album.Pages); i++ {
			if album.Pages[i].Visible {
				g.goTo(i)
				break
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		// go to the last visible page
		for i := len(album.Pages) - 1; i > 0; i-- {
			if album.Pages[i].Visible {
				g.goTo(i)
				break
			}
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.Zoom = !g.Zoom
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		album.Pages[album.CurrentPage].RotateLeft()
		g.needsRefresh = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		album.Pages[album.CurrentPage].RotateRight()
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		album.GrayScale = !album.GrayScale
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.preferences.RemoveBorders = !g.preferences.RemoveBorders
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		album.Reset()
		g.ClearCache()
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		album.Pages[album.CurrentPage].RotationAngle -= 0.05
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		album.Pages[album.CurrentPage].RotationAngle += 0.05
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyKPDivide) {
		album.Pages[album.CurrentPage].RotationAngle = 0
		album.Pages[album.CurrentPage].Top = 0
		album.Pages[album.CurrentPage].Bottom = 0
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || inpututil.IsKeyJustPressed(ebiten.KeyF) {

		if !g.preferences.FullScreen {
			// save the size of the window
			g.preferences.WindowedSize.w, g.preferences.WindowedSize.h = ebiten.WindowSize()
		}
		g.preferences.FullScreen = !g.preferences.FullScreen
		g.InitFullScreen()
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		AppQuit(g.preferences)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyShift) {
		if album.Pages[album.CurrentPage].Position == SinglePage && album.CurrentPage < len(album.Pages)-1 {
			// only if we have a visible page after the current one
			for i := album.CurrentPage + 1; i < len(album.Pages); i++ {
				if album.Pages[i].Visible {
					album.Pages[album.CurrentPage].Position = LeftPage
					album.Pages[i].Position = RightPage
					break
				}
			}
		} else if album.Pages[album.CurrentPage].Position == LeftPage {
			album.Pages[album.CurrentPage].Position = SinglePage
			for i := album.CurrentPage + 1; i < len(album.Pages); i++ {
				if album.Pages[i].Visible {
					album.Pages[i].Position = SinglePage
					break
				}
			}
		}
		g.needsRefresh = true
	}

	return g.refresh()
}

func (g *Gomics) Draw(screen *ebiten.Image) {

	if g.fatalErr != nil {
		ebitenutil.DebugPrintAt(screen, g.fatalErr.Error(), 0, 45)
		return
	}

	if len(g.prominentColors) > 1 {
		backw := g.size.w / len(g.prominentColors)
		x := 0
		for i := 0; i < len(g.prominentColors); i++ {
			backImg := ebiten.NewImage(backw, g.size.h)
			backImg.Fill(g.prominentColors[i])
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(0))
			screen.DrawImage(backImg, op)
			x += backw
		}
	} else {
		screen.Fill(g.prominentColors[0])
	}

	tx := 0
	ty := 0

	width, height := g.currentImage.Size()

	tx = (g.size.w - width) / 2
	ty = (g.size.h - height) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(tx), float64(ty))

	// op.ColorM.ChangeHSV(1.0, 1.0, 1.0)

	screen.DrawImage(g.currentImage, op)

	if g.infoDisplay {
		message := fmt.Sprintf("%0.2f TPS\n%d %%\nscale %.2f\nangle %f", ebiten.CurrentTPS(), album.CurrentPage*100/len(album.Pages), album.Pages[album.CurrentPage].scale, album.Pages[album.CurrentPage].RotationAngle)
		ebitenutil.DebugPrint(screen, message)
	}
	/*
		for i := 0; i < len(g.messages); i++ {
			text.Draw(screen, g.messages[i].Message, font, 0, 0, color.White)
		}
	*/

}

func (g *Gomics) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Return screen size
	// return g.size.w, g.size.h
	return outsideWidth, outsideHeight
}

var comicBook files.ComicBookArchive
var configFolder string
var archiveFile string

type Size struct {
	w, h int
}

var album Album

func (g *Gomics) NextPage() bool {
	for i := album.CurrentPage + 1; i < len(album.Pages); i++ {
		if album.Pages[i].Visible && album.Pages[i].Position != RightPage {
			err := g.goTo(i)
			if err != nil {
				log.Fatal(err)
			}
			return true
		}
	}
	return false
}

func (g *Gomics) PreviousPage() bool {
	for i := album.CurrentPage - 1; i >= 0; i-- {
		if album.Pages[i].Visible && album.Pages[i].Position != RightPage {
			err := g.goTo(i)
			if err != nil {
				log.Fatal(err)
			}
			return true
		}
	}
	return false
}

func AppQuit(preferences Preferences) {
	saveConfiguration(preferences)
	os.Exit(0)
}

func (g *Gomics) preparePage(pageData *PageData) error {

	pageData.mu.Lock()
	defer pageData.mu.Unlock()

	if !pageData.Visible || pageData.ebitenImage != nil {
		// image was already prepared
		return nil
	}

	var err error
	if pageData.rawImage == nil {
		pageData.rawImage, err = comicBook.ReadEntry(pageData.FileName)
		if err != nil {
			log.Printf("Error reading page %s - %s\n", pageData.FileName, err.Error())
			return err
		}
	}

	img := pageData.rawImage
	if pageData.Rotation != None {
		if pageData.Rotation == Left {
			img = imaging.Rotate90(pageData.rawImage)
		} else if pageData.Rotation == Right {
			img = imaging.Rotate270(pageData.rawImage)
		}
	}

	if album.GrayScale {
		img = imaging.Grayscale(img)
	}

	if pageData.Bottom > 0 || pageData.Top > 0 {
		img = imaging.Crop(img, image.Rectangle{Min: image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y+pageData.Top), Max: image.Pt(img.Bounds().Max.X, img.Bounds().Max.Y-pageData.Bottom)})
	}

	if g.preferences.RemoveBorders {

		cropRect := crop.CropBorders(img)
		// colorcrop requires the image to implement this interface to work
		_, ok := interface{}(img).(interface {
			SubImage(r image.Rectangle) image.Image
		})
		if ok {
			img = img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(cropRect)
		}
	}

	if pageData.RotationAngle != 0 {
		img = imaging.Rotate(img, pageData.RotationAngle, color.RGBA{255, 255, 255, 255})
	}

	var imageFilter imaging.ResampleFilter
	if g.preferences.Filter == LANCZOS {
		imageFilter = imaging.Lanczos
	} else {
		imageFilter = imaging.NearestNeighbor
	}

	maxBounds := img.Bounds().Max
	ratio := float64(maxBounds.X) / float64(maxBounds.Y)
	if maxBounds.Y > maxBounds.X {
		sizeY := g.size.h
		if maxBounds.Y < g.size.h {
			sizeY = maxBounds.Y
		}
		pageData.scale = float64(sizeY) / float64(maxBounds.Y)
		img = imaging.Resize(img, 0, sizeY, imageFilter)
	} else {
		sizeX := g.size.w
		if maxBounds.X < g.size.w {
			sizeX = maxBounds.X
		}
		sizeY := float64(sizeX) * ratio
		if sizeY > float64(g.size.h) {
			sizeX = int(float64(g.size.h) * ratio)
		}
		pageData.scale = float64(sizeX) / float64(maxBounds.X)
		img = imaging.Resize(img, sizeX, 0, imageFilter)
	}

	if !pageData.ProminentCalculated {
		// K=1 seems to work better than 3 for us
		kmeans, err := prominentcolor.KmeansWithAll(1, img, prominentcolor.ArgumentDefault|prominentcolor.ArgumentNoCropping, prominentcolor.DefaultSize, []prominentcolor.ColorBackgroundMask{prominentcolor.MaskWhite})
		if err == nil {
			pageData.ProminentColor = color.RGBA{
				R: uint8(kmeans[0].Color.R),
				G: uint8(kmeans[0].Color.G),
				B: uint8(kmeans[0].Color.B),
				A: 255,
			}
		}
		pageData.ProminentCalculated = true
	}

	pageData.ebitenImage = ebiten.NewImageFromImage(img)

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

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Need param")
	}

	var icons []image.Image
	image, _, _ := image.Decode(bytes.NewReader(resources.Gogoreader_png))
	icons = append(icons, image)
	ebiten.SetWindowIcon(icons)

	archiveFile = os.Args[1]
	log.Println("Opening file ", archiveFile)

	var err error

	gomics := &Gomics{}

	log.Printf("Loading %s\n", archiveFile)
	comicBook, err = files.FromFile(archiveFile)
	if err != nil {
		gomics.fatalErr = err
	} else {
		defer comicBook.Close()
		err = comicBook.Init()
		if err != nil {
			gomics.fatalErr = err
		}
	}

	ebiten.SetWindowTitle(archiveFile)

	if gomics.fatalErr != nil {
		gomics.preferences.FullScreen = false
		gomics.preferences.WindowedSize.w = 500
		gomics.preferences.WindowedSize.h = 100
	} else {
		gomics.preferences, err = readConfiguration(comicBook.GetMD5())
		if err != nil {
			gomics.fatalErr = err
		} else {
			ebiten.SetWindowResizable(true)
		}
	}

	ebiten.SetWindowSize(gomics.preferences.WindowedSize.w, gomics.preferences.WindowedSize.h)
	gomics.size.w, gomics.size.h = ebiten.WindowSize()
	ebiten.SetRunnableOnUnfocused(true)
	gomics.InitFullScreen()
	gomics.needsRefresh = true

	if err := ebiten.RunGame(gomics); err != nil {
		panic(err)
	}

	if gomics.fatalErr != nil {
		os.Exit(-1)
	} else {
		AppQuit(gomics.preferences)
	}
}

func (g *Gomics) goTo(newImageIndex int) error {
	if newImageIndex == album.CurrentPage {
		return nil
	}
	album.CurrentPage = newImageIndex
	g.needsRefresh = true
	return nil
}

func (g *Gomics) ClearCache() {
	for index := 0; index < len(album.Pages); index++ {
		album.Pages[index].ebitenImage = nil
	}
	g.needsRefresh = true
}

func (g *Gomics) processPage(pageNum int, currentImages []*ebiten.Image, prominentColors []color.RGBA) ([]*ebiten.Image, []color.RGBA, error) {
	pageData := &album.Pages[pageNum]
	err := g.preparePage(pageData)
	if err != nil {
		return nil, nil, err
	}
	currentImages = append(currentImages, pageData.ebitenImage)
	prominentColors = append(prominentColors, pageData.ProminentColor)

	return currentImages, prominentColors, nil
}

func (g *Gomics) refresh() error {

	if !g.needsRefresh {
		return nil
	}
	g.needsRefresh = false

	album.Pages[album.CurrentPage].ebitenImage = nil

	var currentImages []*ebiten.Image
	g.prominentColors = g.prominentColors[:0]

	var err error

	currentImages, g.prominentColors, err = g.processPage(album.CurrentPage, currentImages, g.prominentColors)
	if err != nil {
		return err
	}
	if album.Pages[album.CurrentPage].Position == LeftPage {
		for i := album.CurrentPage + 1; i < len(album.Pages); i++ {
			if album.Pages[i].Visible && album.Pages[i].Position == RightPage {
				currentImages, g.prominentColors, err = g.processPage(i, currentImages, g.prominentColors)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	if len(currentImages) > 1 {
		totalWidth := 0
		maxHeight := 0
		for _, img := range currentImages {
			width, height := img.Size()
			if height > maxHeight {
				maxHeight = height
			}
			totalWidth += width
		}

		g.currentImage = ebiten.NewImage(totalWidth, maxHeight)

		tx := 0
		for _, img := range currentImages {
			width, height := img.Size()
			ty := (maxHeight - height) / 2
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(tx), float64(ty))
			g.currentImage.DrawImage(img, opts)
			tx += width
		}
	} else {
		g.currentImage = currentImages[0]
	}

	for i := album.CurrentPage + 1; i < len(album.Pages); i++ {
		if album.Pages[i].Position == LeftPage || album.Pages[i].Position == SinglePage {
			// prepare next page in the background
			pageData := &album.Pages[i]
			go g.preparePage(pageData)
			if pageData.Position == LeftPage {
				for j := i + 1; j < len(album.Pages); j++ {
					if album.Pages[j].Visible {
						go g.preparePage(&album.Pages[j])
						break
					}
				}
			}
			break
		}
	}

	if album.CurrentPage > 1 {
		// remove old images from cache
		for index := 0; index < album.CurrentPage; index++ {
			album.Pages[index].ebitenImage = nil
		}
	}

	return nil
}
