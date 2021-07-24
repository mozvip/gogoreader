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
}

func (a *Album) GetConfigurationFile(configFolder string) string {
	return path.Join(configFolder, a.MD5+".yml")
}

type AlbumPreferences struct {
	MD5         string
	CurrentPage int
	Pages       []PagePreferences
}
