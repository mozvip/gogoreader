package main

import "path"

type Rotation uint8

const (
	None Rotation = iota
	Left
	Right
)

type Album struct {
	MD5              string
	CurrentPageIndex int
	Pages            []*PageData
	Images           []*ImageData `json:"-"`
	GrayScale        bool
}

func (a *Album) GetCurrentPage() *PageData {
	return a.Pages[a.CurrentPageIndex]
}

func (a *Album) GetConfigurationFile(configFolder string) string {
	return path.Join(configFolder, a.MD5+".yml")
}

func (a *Album) Reset() {
	for i := 0; i < len(a.Pages); i++ {
		a.Pages[i].Reset()
		a.Pages[i].BackgroundColors = nil
	}
	for _, i := range a.Images {
		i.Visible = true
		i.Rotation = None
	}
	a.CurrentPageIndex = 0
}
