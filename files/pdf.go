package files

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type PDFComicBook struct {
	FileWithMD5
	f         *os.File
	Pages     []string
	pdfReader *model.PdfReader
}

func (P *PDFComicBook) Close() {
	P.f.Close()
}

func (P *PDFComicBook) List() ([]string, error) {
	return P.Pages, nil
}

func (P *PDFComicBook) ReadEntry(fileName string) (image.Image, error) {

	var index int
	fmt.Sscanf(fileName, "Page %d", &index)

	page, err := P.pdfReader.GetPage(index)
	if err != nil {
		return nil, err
	}
	pextract, err := extractor.New(page)
	if err != nil {
		return nil, err
	}
	pimages, err := pextract.ExtractPageImages(nil)
	if err != nil {
		return nil, err
	}

	img := pimages.Images[0]
	gimg, err := img.Image.ToGoImage()
	return gimg, err
}

func (P *PDFComicBook) GetMD5() string {
	return P.MD5
}

func (P *PDFComicBook) Init() (err error) {
	P.f, err = os.Open(P.FileName)
	if err != nil {
		return err
	}

	P.pdfReader, err = model.NewPdfReader(P.f)
	if err != nil {
		return err
	}

	isEncrypted, err := P.pdfReader.IsEncrypted()
	if err != nil {
		return err
	}

	// Try decrypting with an empty one.
	if isEncrypted {
		auth, err := P.pdfReader.Decrypt([]byte(""))
		if err != nil {
			// Encrypted and we cannot do anything about it.
			return err
		}
		if !auth {
			return errors.New("PDF file is encrypted")
		}
	}

	numPages, err := P.pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	for i := 0; i < numPages; i++ {
		P.Pages = append(P.Pages, fmt.Sprintf("Page %03d", i+1))
	}

	return nil
}
