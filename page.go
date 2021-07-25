package main

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type PagePreferences struct {
	AlbumPreferencesID uint

	Rotation Rotation
	Position Position
	Visible  bool

	Top    int
	Bottom int
}

type PageData struct {
	FileName            string
	Rotation            Rotation
	Position            Position
	Visible             bool
	ProminentColor      color.RGBA
	ProminentCalculated bool

	Top    int
	Bottom int

	rawImage image.Image
	imgData  *ebiten.Image
	mu       sync.Mutex
}

func (p *PageData) RotateRight() {
	if p.Rotation == None {
		p.Rotation = Right
	} else if p.Rotation == Left {
		p.Rotation = None
	}
	p.imgData = nil
	p.ProminentCalculated = false
}

func (p *PageData) RotateLeft() {
	if p.Rotation == None {
		p.Rotation = Left
	} else if p.Rotation == Right {
		p.Rotation = None
	}
	p.imgData = nil
	p.ProminentCalculated = false
}
