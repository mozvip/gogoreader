package main

import (
	"path"
)

type ImageFilter uint

const (
	LANCZOS          ImageFilter = 0
	NEAREST_NEIGHBOR ImageFilter = 1
)

type Vector2Int struct {
	X int32
	Y int32
}

func NewVector2Int(X int, Y int) Vector2Int {
	return Vector2Int{X: int32(X), Y: int32(Y)}
}

type Preferences struct {
	FullScreen    bool
	RemoveBorders bool
	Filter        ImageFilter
	WindowedSize  Vector2Int
}

func NewPreferences() Preferences {
	preferences := Preferences{}
	preferences.Filter = LANCZOS
	return preferences
}

func getGlobalConfigurationFile() string {
	return path.Join(configFolder, "config.yml")
}
