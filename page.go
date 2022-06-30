package main

import (
	"sync"

	"github.com/faiface/pixel"
)

type ImageData struct {
	FileName string
	Visible  bool
	Rotation Rotation

	// cropping
	Top    int
	Bottom int
	Left   int
	Right  int
}

type ViewData struct {
	mu sync.Mutex

	Images           []*ImageData
	RotationAngle    float64
	BackgroundColors []pixel.RGBA
	RemoveBorders    bool
	bordersOverride  bool

	imageSprites []*pixel.Sprite

	totalWidth float64
	maxHeight  float64
}

func (v *ViewData) updateSize() {

	var totalWidth, maxHeight float64
	for _, sprite := range v.imageSprites {
		spriteW, spriteH := sprite.Frame().W(), sprite.Frame().H()
		totalWidth += spriteW
		if spriteH > maxHeight {
			maxHeight = spriteH
		}
	}
	v.totalWidth = totalWidth
	v.maxHeight = maxHeight
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
	p.updateSize()
}
