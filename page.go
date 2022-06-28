package main

import (
	"image/color"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ImageData struct {
	FileName string
	Visible  bool
	Rotation Rotation

	// cropping
	Top    float32
	Bottom float32
	Left   float32
	Right  float32
}

type ViewData struct {
	mu sync.Mutex

	Images           []*ImageData
	RotationAngle    float64
	BackgroundColors []color.RGBA
	RemoveBorders    bool
	bordersOverride  bool

	TotalWidth uint32
	MaxHeight  uint32

	images   []*rl.Image
	textures []rl.Texture2D
}

func (p *ViewData) RotateRight() {
	for i := 0; i < len(p.Images); i++ {
		if p.Images[i].Rotation == None {
			p.Images[i].Rotation = Right
		} else if p.Images[i].Rotation == Left {
			p.Images[i].Rotation = None
		}
	}
	p.Reset()
}

func (p *ViewData) RotateLeft() {
	for i := 0; i < len(p.Images); i++ {
		if p.Images[i].Rotation == None {
			p.Images[i].Rotation = Left
		} else if p.Images[i].Rotation == Right {
			p.Images[i].Rotation = None
		}
	}
	p.Reset()
}

func (p *ViewData) ToggleBorder(globalSetting bool) {
	p.RemoveBorders = !p.RemoveBorders
	p.bordersOverride = p.RemoveBorders != globalSetting
}

func (p *ViewData) Reset() {
	p.bordersOverride = false
	for i := 0; i < len(p.Images); i++ {
		p.Images[i].Top = 0
		p.Images[i].Bottom = 0
		p.Images[i].Left = 0
		p.Images[i].Right = 0
	}

}
