package files

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"image"
	"image/draw"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/image/webp"
)

type FileWithMD5 struct {
	FileName string
	MD5      string
}

type ComicBookArchive interface {
	Close()
	List() ([]string, error)
	ReadEntry(fileName string) (*image.NRGBA, error)
	GetMD5() string
	Init() error
}

var imageCache map[string]*image.NRGBA

func CreateImageFromReader(fileName string, reader io.Reader) (*image.NRGBA, error) {
	if imageCache == nil {
		imageCache = make(map[string]*image.NRGBA)
	}
	src, hasKey := imageCache[fileName]
	if hasKey {
		return src, nil
	}
	var e error
	var loadedImage image.Image
	if strings.HasSuffix(fileName, "webp") {
		loadedImage, e = webp.Decode(reader)
	} else {
		loadedImage, _, e = image.Decode(reader)
	}

	if e != nil {
		return nil, e
	}

	var rgbaImg *image.NRGBA
	var ok bool
	if rgbaImg, ok = loadedImage.(*image.NRGBA); !ok {
		b := loadedImage.Bounds()
		rgbaImg = image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgbaImg, rgbaImg.Bounds(), loadedImage, b.Min, draw.Src)
	}
	imageCache[fileName] = rgbaImg
	return rgbaImg, e
}

func newZippedComicBook(MD5 string, fileName string) (*ZippedComicBook, error) {
	zipReader, err := zip.OpenReader(fileName)
	if err != nil {
		return nil, err
	}
	return &ZippedComicBook{FileWithMD5: FileWithMD5{MD5: MD5}, zip: zipReader}, nil
}

func newRaredComicBook(md5 string, fileName string) (*RaredComicBook, error) {
	return &RaredComicBook{FileWithMD5: FileWithMD5{FileName: fileName, MD5: md5}, contents: nil}, nil
}

func newPDFComicBook(md5 string, fileName string) (ComicBookArchive, error) {
	return &PDFComicBook{FileWithMD5: FileWithMD5{FileName: fileName, MD5: md5}}, nil
}

// FromFile creates a new ComicBookArchive from the given file name
func FromFile(fileName string) (ComicBookArchive, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	log.Printf("Computing MD5 for %s\n", fileName)
	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return nil, err
	}
	fileMD5 := fmt.Sprintf("%x", h.Sum(nil))

	log.Printf("File MD5 is %s\n", fileMD5)

	lower := strings.ToLower(fileName)

	if IsValidRar(fileName) {
		return newRaredComicBook(fileMD5, fileName)
	} else if strings.HasSuffix(lower, ".pdf") {
		return newPDFComicBook(fileMD5, fileName)
	} else {
		return newZippedComicBook(fileMD5, fileName)
	}
}
