package main

//go:generate binclude

import (
	"bytes"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/lu4p/binclude"
	"github.com/mozvip/gomics/files"
	"github.com/nxshock/colorcrop"
)

type Gomics struct {
	currentImages []*ebiten.Image
	size          Size
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

func (g *Gomics) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		album.Pages[album.CurrentPage].Top++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		album.Pages[album.CurrentPage].Bottom++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.NextPage()
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
		g.goTo(0) // FIXME : First visible may not be 0
	}

	lastImageIndex := len(album.Pages) - 1
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		g.goTo(lastImageIndex)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		album.Pages[album.CurrentPage].RotateLeft()
		g.refresh()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		album.Pages[album.CurrentPage].RotateRight()
		g.refresh()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		preferences.GrayScale = !preferences.GrayScale
		g.ClearCache()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		preferences.RemoveBorders = !preferences.RemoveBorders
		g.ClearCache()
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
		if album.Pages[album.CurrentPage].Position == SinglePage {
			album.Pages[album.CurrentPage].Position = LeftPage
			album.Pages[album.CurrentPage+1].Position = RightPage
		} else if album.Pages[album.CurrentPage].Position == LeftPage {
			album.Pages[album.CurrentPage].Position = SinglePage
			album.Pages[album.CurrentPage+1].Position = SinglePage
		}
		g.refresh()
	}

	return nil

}

func (g *Gomics) Draw(screen *ebiten.Image) {

	pageData := album.Pages[album.CurrentPage]
	ebiten.SetWindowTitle(pageData.FileName)
	if pageData.ProminentCalculated {
		screen.Fill(pageData.ProminentColor)
	}

	totalWidth := 0
	for _, img := range g.currentImages {
		width, _ := img.Size()
		totalWidth += width
	}

	currentX := g.size.w/2 - totalWidth/2
	for _, img := range g.currentImages {
		width, height := img.Size()

		// pageData.Bottom - pageData.Top

		top := 0
		if height < width {
			top = (g.size.h - height) / 2
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(currentX), float64(top))
		/*
				SourceRect: &image.Rectangle{
					Min: image.Point{Y: pageData.Top},
					Max: image.Point{X: width, Y: height - pageData.Bottom},
				},
			}
		*/
		screen.DrawImage(img, op)
		currentX += width
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

type Preferences struct {
	FullScreen    bool
	GrayScale     bool
	RemoveBorders bool
	WindowedSize  Size
}

var preferences Preferences

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

func (g *Gomics) preparePage(pageData *PageData) (err error) {

	if pageData.imgData != nil {
		// image was already prepared
		return nil
	}

	if !pageData.Visible {
		return nil
	}

	pageData.mu.Lock()
	defer pageData.mu.Unlock()

	if pageData.rawImage == nil {
		pageData.rawImage, err = comicBook.ReadEntry(pageData.FileName)
		if err != nil {
			return err
		}
	}

	img := pageData.rawImage
	if preferences.RemoveBorders {
		img = colorcrop.Crop(
			img,                            // for source image
			color.RGBA{255, 255, 255, 255}, // crop white border
			0.5)                            // with 50% thresold
	}

	if pageData.Rotation != None {
		if pageData.Rotation == Left {
			img = imaging.Rotate90(img)
		} else if pageData.Rotation == Right {
			img = imaging.Rotate270(img)
		}
	}

	imageBounds := img.Bounds().Max
	if imageBounds.Y > imageBounds.X {
		sizeY := g.size.h
		if imageBounds.Y < g.size.h {
			sizeY = imageBounds.Y
		}
		img = imaging.Resize(img, 0, sizeY, imaging.Lanczos)
	} else {
		sizeX := g.size.w
		if imageBounds.X < g.size.w {
			sizeX = imageBounds.X
		}
		img = imaging.Resize(img, sizeX, 0, imaging.Lanczos)
	}

	if preferences.GrayScale {
		img = imaging.Grayscale(img)
	}

	pageData.imgData = ebiten.NewImageFromImage(img)
	if !pageData.ProminentCalculated {
		kmeans, err := prominentcolor.Kmeans(img)
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

	return err
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
	comicBook.Init()

	err = readConfiguration(comicBook.GetMD5())
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(preferences.WindowedSize.w, preferences.WindowedSize.h)
	ebiten.SetWindowTitle("gomics")
	gomics := &Gomics{}
	gomics.size.w, gomics.size.h = ebiten.WindowSize()

	gomics.InitFullScreen()
	gomics.refresh()

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
	return g.refresh()
}

func (g *Gomics) ClearCache() {
	for index := 0; index < len(album.Pages); index++ {
		album.Pages[index].imgData = nil
	}
	g.refresh()
}

func (g *Gomics) refresh() error {

	g.currentImages = g.currentImages[:0]
	var nextPage int
	for index := album.CurrentPage; index < len(album.Pages); index++ {
		pageData := &album.Pages[index]
		err := g.preparePage(pageData)
		if err != nil {
			return err
		}
		g.currentImages = append(g.currentImages, pageData.imgData)
		if pageData.Position != LeftPage {
			nextPage = index + 1
			break
		}
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

	return nil
}
