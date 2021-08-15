package main

import (
	"path"

	"github.com/hajimehoshi/ebiten/v2"
)

type Preferences struct {
	FullScreen    bool
	GrayScale     bool
	RemoveBorders bool
	Filter        ebiten.Filter
	WindowedSize  Size
}

var preferences Preferences

func NewPreferences() Preferences {
	preferences := Preferences{}
	preferences.Filter = ebiten.FilterLinear
	return preferences
}

func getGlobalConfigurationFile() string {
	return path.Join(configFolder, "config.yml")
}
