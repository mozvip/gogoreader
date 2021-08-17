package main

import (
	"path"

	"github.com/disintegration/imaging"
)

type Preferences struct {
	FullScreen    bool
	RemoveBorders bool
	Filter        imaging.ResampleFilter
	WindowedSize  Size
}

func NewPreferences() Preferences {
	preferences := Preferences{}
	preferences.Filter = imaging.Lanczos
	return preferences
}

func getGlobalConfigurationFile() string {
	return path.Join(configFolder, "config.yml")
}
