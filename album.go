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
	CurrentViewIndex int
	Views            []*ViewData
	Images           []*ImageData `json:"-"`
	GrayScale        bool
	RemoveBorders    bool
}

func (a *Album) GetCurrentView() *ViewData {
	return a.Views[a.CurrentViewIndex]
}

func (a *Album) GetConfigurationFile(configFolder string) string {
	return path.Join(configFolder, a.MD5+".yml")
}

func (a *Album) Reset() {
	for i := 0; i < len(a.Views); i++ {
		a.Views[i].Reset()
		a.Views[i].BackgroundColors = nil
	}
	for _, i := range a.Images {
		i.Visible = true
		i.Rotation = None
	}
	a.CurrentViewIndex = 0
}
