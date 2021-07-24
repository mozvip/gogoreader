package files

import (
	"image"
	"io"

	"github.com/nwaples/rardecode"
)

type RaredComicBook struct {
	FileWithMD5
	contents []string
}

func (z *RaredComicBook) Close() {
}

func (z *RaredComicBook) GetMD5() string {
	return z.MD5
}

func (z *RaredComicBook) List() ([]string, error) {
	return z.contents, nil
}

func (z *RaredComicBook) ReadEntry(fileName string) (image.Image, error) {

	var err error
	var header *rardecode.FileHeader
	var archive *rardecode.ReadCloser

	archive, err = rardecode.OpenReader(z.FileName, "")
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	for err != io.EOF {
		header, err = archive.Next()
		if err == nil {
			if header.Name == fileName && header.UnPackedSize > 0 {
				return CreateImageFromReader(fileName, archive)
			}
		} else {
			if err != io.EOF {
				return nil, err
			}
		}
	}

	return nil, err
}

func (z *RaredComicBook) Init() error {
	var err error
	var header *rardecode.FileHeader
	var archive *rardecode.ReadCloser

	archive, err = rardecode.OpenReader(z.FileName, "")
	if err != nil {
		return err
	}
	defer archive.Close()

	for err != io.EOF {
		header, err = archive.Next()
		if err == nil {
			if header.IsDir {
				continue
			}
			z.contents = append(z.contents, header.Name)
		} else {
			if err != io.EOF {
				return err
			}
		}

	}
	return nil
}
