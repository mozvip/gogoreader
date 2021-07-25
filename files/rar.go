package files

import (
	"image"
	"io"

	"github.com/nwaples/rardecode"
)

type RaredComicBook struct {
	FileWithMD5
	contents           []string
	archive            *rardecode.ReadCloser
	header             *rardecode.FileHeader
	currentHeaderIndex int
}

func (z *RaredComicBook) Close() {
	z.archive.Close()
}

func (z *RaredComicBook) GetMD5() string {
	return z.MD5
}

func (z *RaredComicBook) List() ([]string, error) {
	return z.contents, nil
}

func (z *RaredComicBook) reload() error {
	var err error

	z.currentHeaderIndex = -1

	// reopen the archive at the beginning and store it in the struct
	z.archive, err = rardecode.OpenReader(z.FileName, "")
	if err != nil {
		return nil
	}
	z.header, err = z.archive.Next()
	if err != nil {
		return nil
	}

	z.currentHeaderIndex = 0
	return nil
}

func (z *RaredComicBook) ReadEntry(fileName string) (image.Image, error) {

	for index, v := range z.contents {
		if fileName == v {
			if index < z.currentHeaderIndex {
				// we need to reload the rar file
				z.reload()
				break
			}
		}
	}

	var err error
	for err != io.EOF {
		if z.header.Name == fileName && z.header.UnPackedSize > 0 {
			return CreateImageFromReader(fileName, z.archive)
		}
		z.header, err = z.archive.Next()
		z.currentHeaderIndex++
		if err != nil && err != io.EOF {
			break
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

	return z.reload()
}
