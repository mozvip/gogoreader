package main

import "path"

type Rotation uint8

const (
	None Rotation = iota
	Left
	Right
)

type Position uint8

const (
	SinglePage Position = iota
	LeftPage
	RightPage
)

type Album struct {
	Path        string
	MD5         string
	CurrentPage int
	Pages       []PageData
	GrayScale   bool
}

func (a *Album) GetConfigurationFile(configFolder string) string {
	return path.Join(configFolder, a.MD5+".yml")
}

func (a *Album) Reset() {
	for i := 0; i < len(a.Pages); i++ {
		a.Pages[i].Visible = true
		a.Pages[i].Bottom = 0
		a.Pages[i].Top = 0
		a.Pages[i].Rotation = None
		a.Pages[i].Position = SinglePage
	}
	a.CurrentPage = 0
}
