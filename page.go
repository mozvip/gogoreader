package main

import (
	"sync"

	"github.com/faiface/pixel"
)

type ImageData struct {
	FileName string
	Visible  bool
	Rotation Rotation

	Top    float64
	Bottom float64
	Left   float64
	Right  float64
}

type PageData struct {
	mu sync.Mutex

	Images           []*ImageData
	RotationAngle    float64
	BackgroundColors []pixel.RGBA

	imageSprites []*pixel.Sprite
}

func (p *PageData) RotateRight() {
	for i := 0; i < len(p.Images); i++ {
		if p.Images[i].Rotation == None {
			p.Images[i].Rotation = Right
		} else if p.Images[i].Rotation == Left {
			p.Images[i].Rotation = None
		}
	}
	p.Reset()
}

func (p *PageData) RotateLeft() {
	for i := 0; i < len(p.Images); i++ {
		if p.Images[i].Rotation == None {
			p.Images[i].Rotation = Left
		} else if p.Images[i].Rotation == Right {
			p.Images[i].Rotation = None
		}
	}
	p.Reset()
}

func (p *PageData) Reset() {
	for i := 0; i < len(p.Images); i++ {
		p.Images[i].Top = 0
		p.Images[i].Bottom = 0
		p.Images[i].Left = 0
		p.Images[i].Right = 0
	}

}
