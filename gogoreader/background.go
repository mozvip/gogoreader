package gogoreader

import (
	"image"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func GetPixelColor(img *image.NRGBA, x uint32, y uint32) (r, g, b uint8) {
	r1, g1, b1, _ := img.At(int(x), int(y)).RGBA()
	return uint8(r1 >> 8), uint8(g1 >> 8), uint8(b1 >> 8)
}

func AverageColor(pictureData *image.NRGBA, rect rl.Rectangle) color.RGBA {
	step := float32(3)
	var count, sr, sg, sb uint32
	for x := rect.X; x < rect.X+rect.Width; x += step {
		intX := uint32(x)
		for y := rect.Y; y < rect.Y+rect.Height; y += step {
			r, g, b := GetPixelColor(pictureData, intX, uint32(y))
			if r > 240 && g > 240 && b > 240 {
				// ignore white & almost white pixels
				continue
			}
			sr += uint32(r)
			sg += uint32(g)
			sb += uint32(b)
			count++
		}
	}
	if count > 0 {
		return color.RGBA{R: uint8(sr / count), G: uint8(sg / count), B: uint8(sb / count), A: 255}
	}

	return color.RGBA{R: uint8(0), G: uint8(0), B: uint8(0), A: 255}
}

func ProminentColor(pictureData *image.NRGBA, rect rl.Rectangle) color.RGBA {
	step := float32(3.0)

	colorsCount := make(map[color.RGBA]uint)
	currentMax := uint(0)
	prominentColor := color.RGBA{R: uint8(0), G: uint8(0), B: uint8(0), A: 255}
	for x := rect.X; x < rect.X+rect.Width; x += step {
		intX := uint32(x)
		for y := rect.Y; y < rect.Y+rect.Height; y += step {

			r, g, b := GetPixelColor(pictureData, intX, uint32(y))
			if r > 240 && g > 240 && b > 240 {
				// ignore white pixels
				continue
			}
			if r < 15 && g < 15 && b < 15 {
				// ignore black pixels
				continue
			}
			// remove last bit of precision
			r = r >> 1 << 1
			g = g >> 1 << 1
			b = b >> 1 << 1
			color := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			count, hasKey := colorsCount[color]
			if hasKey {
				colorsCount[color] = count + 1
			} else {
				colorsCount[color] = 1
			}
			if colorsCount[color] > currentMax {
				prominentColor = color
				currentMax = colorsCount[color]
			}
		}
	}
	return prominentColor
}
