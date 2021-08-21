package files

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"image"
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
	ReadEntry(fileName string) (image.Image, error)
	GetMD5() string
	Init() error
}

func CreateImageFromReader(fileName string, reader io.Reader) (image.Image, error) {
	if strings.HasSuffix(fileName, "webp") {
		return webp.Decode(reader)
	} else {
		img, _, e := image.Decode(reader)
		return img, e
	}
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
	if strings.HasSuffix(lower, ".pdf") {
		return newPDFComicBook(fileMD5, fileName)
	} else if strings.HasSuffix(lower, ".cbz") || strings.HasSuffix(lower, ".zip") {
		return newZippedComicBook(fileMD5, fileName)
	} else if strings.HasSuffix(lower, ".cbr") || strings.HasSuffix(lower, ".rar") {
		return newRaredComicBook(fileMD5, fileName)
	}
	return nil, fmt.Errorf("unable to determine type of archive for file %s", fileName)
}
