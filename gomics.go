package main

//go:generate binclude

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/lu4p/binclude"
	"github.com/mozvip/gomics/files"
	"github.com/nxshock/colorcrop"
)

type Gomics struct {
	currentImage    *ebiten.Image
	prominentColors []color.RGBA
	size            Size
	needsRefresh    bool
	infoDisplay     bool
}

func (g *Gomics) InitFullScreen() {
	ebiten.SetFullscreen(preferences.FullScreen)
	if !preferences.FullScreen {
		// restore the size of the window
		g.size = preferences.WindowedSize
	} else {
		g.size.w, g.size.h = ebiten.ScreenSizeInFullscreen()
	}
}

func (g *Gomics) toggleInfoDisplay() {
	g.infoDisplay = !g.infoDisplay
}

func (g *Gomics) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		g.toggleInfoDisplay()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		album.Pages[album.CurrentPage].Top++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		album.Pages[album.CurrentPage].Bottom++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.NextPage()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		preferences.Filter = ebiten.FilterLinear
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		preferences.Filter = ebiten.FilterNearest
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

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		album.Pages[album.CurrentPage].RotateLeft()
		g.needsRefresh = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		album.Pages[album.CurrentPage].RotateRight()
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		// FIXME : move at the album level, not global
		preferences.GrayScale = !preferences.GrayScale
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		preferences.RemoveBorders = !preferences.RemoveBorders
		g.ClearCache()
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
		g.needsRefresh = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || inpututil.IsKeyJustPressed(ebiten.KeyF) {

		if !preferences.FullScreen {
			// save the size of the window
			preferences.WindowedSize.w, preferences.WindowedSize.h = ebiten.WindowSize()
		}
		preferences.FullScreen = !preferences.FullScreen
		g.InitFullScreen()
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		AppQuit()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyShift) {
		if album.Pages[album.CurrentPage].Position == SinglePage && album.CurrentPage < len(album.Pages)-1 {
			album.Pages[album.CurrentPage].Position = LeftPage
			album.Pages[album.CurrentPage+1].Position = RightPage
		} else if album.Pages[album.CurrentPage].Position == LeftPage {
			album.Pages[album.CurrentPage].Position = SinglePage
			album.Pages[album.CurrentPage+1].Position = SinglePage
		}
		g.needsRefresh = true
	}

	return g.refresh()
}

func (g *Gomics) Draw(screen *ebiten.Image) {

	/*
		w := g.size.w / 2
		if len(g.prominentColors) > 1 {
			left := image.Rectangle{Min: image.Pt(0, 0), Max: image.Pt(w, g.size.h)}
			right := image.Rectangle{Min: image.Pt(w, 0), Max: image.Pt(g.size.w, g.size.h)}
			screen.SubImage(left).Fill(g.prominentColors[0])
			screen.SubImage(right).Fill(g.prominentColors[1])
		}
	*/

	screen.Fill(g.prominentColors[0])

	tx := 0
	ty := 0
	scale := float64(1)

	width, height := g.currentImage.Size()

	xscale := float64(g.size.w) / float64(width)
	yscale := float64(g.size.h) / float64(height)

	if xscale < yscale {
		scale = xscale
		tx = 0
		ty = int(((float64(g.size.h) - (float64(height) * scale)) / float64(2)))
	} else {
		scale = yscale
		ty = 0
		tx = int(((float64(g.size.w) - (float64(width) * scale)) / float64(2)))
	}

	op := &ebiten.DrawImageOptions{}
	//op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(tx), float64(ty))
	op.Filter = preferences.Filter

	// op.ColorM.ChangeHSV(1.0, 1.0, 1.0)

	screen.DrawImage(g.currentImage, op)

	if g.infoDisplay {
		message := fmt.Sprintf("%d %%\nscale %.2f\nangle %f", album.CurrentPage*100/len(album.Pages), scale, album.Pages[album.CurrentPage].RotationAngle)
		ebitenutil.DebugPrint(screen, message)
	}
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

func AppQuit() {
	saveConfiguration()
	os.Exit(0)
}

func (g *Gomics) preparePage(pageData *PageData) error {

	pageData.mu.Lock()
	defer pageData.mu.Unlock()

	if !pageData.Visible || pageData.imgData != nil {
		// image was already prepared
		return nil
	}

	img, err := comicBook.ReadEntry(pageData.FileName)
	if err != nil {
		log.Printf("Error reading page %s - %s\n", pageData.FileName, err.Error())
		return err
	}

	if pageData.Rotation != None {
		if pageData.Rotation == Left {
			img = imaging.Rotate90(img)
		} else if pageData.Rotation == Right {
			img = imaging.Rotate270(img)
		}
	}

	if preferences.GrayScale {
		img = imaging.Grayscale(img)
	}

	if pageData.RotationAngle != 0 {
		img = imaging.Rotate(img, pageData.RotationAngle, color.RGBA{255, 255, 255, 255})
	}

	if preferences.RemoveBorders {
		// colorcrop requires the image to implement this interface to work
		_, ok := interface{}(img).(interface {
			SubImage(r image.Rectangle) image.Image
		})
		if ok {
			img = colorcrop.Crop(
				img,                            // for source image
				color.RGBA{255, 255, 255, 255}, // crop white border
				0.5)                            // with 50% thresold
		}
	}

	maxBounds := img.Bounds().Max
	if maxBounds.Y > maxBounds.X {
		sizeY := g.size.h
		if maxBounds.Y < g.size.h {
			sizeY = maxBounds.Y
		}
		img = imaging.Resize(img, 0, sizeY, imaging.Lanczos)
	} else {
		sizeX := g.size.w
		if maxBounds.X < g.size.w {
			sizeX = maxBounds.X
		}
		img = imaging.Resize(img, sizeX, 0, imaging.Lanczos)
	}

	if !pageData.ProminentCalculated {
		// K=4 seems to work better than 3 for us
		kmeans, err := prominentcolor.KmeansWithAll(3, img, prominentcolor.ArgumentDefault|prominentcolor.ArgumentNoCropping, prominentcolor.DefaultSize, []prominentcolor.ColorBackgroundMask{prominentcolor.MaskWhite, prominentcolor.MaskBlack})
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

	pageData.imgData = ebiten.NewImageFromImage(img)

	return err
}

var logFile *os.File

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
}

func main() {

	binclude.Include("gomics.png")

	if len(os.Args) < 2 {
		log.Fatal("Need param")
	}
	file, errOpen := BinFS.Open("gomics.png")
	if errOpen == nil {
		iconData, err := ioutil.ReadAll(file)
		if err == nil {
			var icons []image.Image
			image, _, _ := image.Decode(bytes.NewReader(iconData))
			icons = append(icons, image)
			ebiten.SetWindowIcon(icons)
		}
	} else {
		panic("Unable to open gomics.png")
	}

	archiveFile = os.Args[1]
	log.Println("Opening file ", archiveFile)

	var err error
	log.Printf("Loading %s\n", archiveFile)
	comicBook, err = files.FromFile(archiveFile)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer comicBook.Close()
	err = comicBook.Init()
	if err != nil {
		panic(err)
	}

	err = readConfiguration(comicBook.GetMD5())
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(preferences.WindowedSize.w, preferences.WindowedSize.h)
	ebiten.SetWindowTitle(archiveFile)
	gomics := &Gomics{}
	gomics.size.w, gomics.size.h = ebiten.WindowSize()

	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowResizable(true)
	gomics.InitFullScreen()
	gomics.needsRefresh = true

	if err := ebiten.RunGame(gomics); err != nil {
		panic(err)
	}
	AppQuit()
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
		album.Pages[index].imgData = nil
	}
	g.needsRefresh = true
}

func (g *Gomics) refresh() error {

	if !g.needsRefresh {
		return nil
	}
	g.needsRefresh = false

	album.Pages[album.CurrentPage].imgData = nil

	var currentImages []*ebiten.Image
	var nextPage int

	g.prominentColors = g.prominentColors[:0]
	for index := album.CurrentPage; index < len(album.Pages); index++ {
		pageData := &album.Pages[index]
		err := g.preparePage(pageData)
		if err != nil {
			return err
		}
		currentImages = append(currentImages, pageData.imgData)
		g.prominentColors = append(g.prominentColors, pageData.ProminentColor)
		if pageData.Position != LeftPage {
			nextPage = index + 1
			break
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

	// prepare next page in the background
	if nextPage >= 0 && nextPage < len(album.Pages)-1 {
		pageData := &album.Pages[nextPage]
		go g.preparePage(pageData)
		if pageData.Position == LeftPage {
			pageData := &album.Pages[nextPage+1]
			go g.preparePage(pageData)
		}
	}

	if album.CurrentPage > 1 {
		// remove old images from cache
		for index := 0; index < album.CurrentPage; index++ {
			album.Pages[index].imgData = nil
		}
	}

	return nil
}
