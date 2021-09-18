package main

import (
	"path"

	"github.com/faiface/pixel"
)

type ImageFilter uint

const (
	LANCZOS          ImageFilter = 0
	NEAREST_NEIGHBOR ImageFilter = 1
)

type Preferences struct {
	FullScreen    bool
	RemoveBorders bool
	Filter        ImageFilter
	WindowedSize  pixel.Vec
}

func NewPreferences() Preferences {
	preferences := Preferences{}
	preferences.Filter = LANCZOS
	return preferences
}

func getGlobalConfigurationFile() string {
	return path.Join(configFolder, "config.yml")
}
