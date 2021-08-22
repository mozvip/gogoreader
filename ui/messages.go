package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

type Message struct {
	Message string
	timeout float64
}

func NewMessage(message string, timeoutInSeconds float64) Message {
	return Message{Message: message, timeout: timeoutInSeconds}
}

func (m *Message) Draw(img *ebiten.Image, fontFace font.Face, x, y int) {
	text.Draw(img, m.Message, fontFace, x, y, color.White)
}
