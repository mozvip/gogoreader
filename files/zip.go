package files

import (
	"archive/zip"
	"fmt"
	"image"
	"io/ioutil"
)

type ZippedComicBook struct {
	FileWithMD5
	zip *zip.ReadCloser
}

func (z *ZippedComicBook) Close() {
	z.zip.Close()
}

func (z *ZippedComicBook) GetMD5() string {
	return z.MD5
}

func (z *ZippedComicBook) ReadEntry(fileName string) (image.Image, error) {
	for _, f := range z.zip.File {
		if f.Name == fileName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			bytes, err := ioutil.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return CreateImage(fileName, bytes)
		}
	}
	return nil, fmt.Errorf("file %s was not found in archive", fileName)
}

func (z *ZippedComicBook) Init() error {
	return nil
}

func (z *ZippedComicBook) List() ([]string, error) {
	result := make([]string, 0)
	for _, f := range z.zip.File {
		result = append(result, f.Name)
	}
	return result, nil
}

