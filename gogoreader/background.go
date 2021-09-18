package gogoreader

import (
	"image"

	"github.com/disintegration/imaging"
)

func AverageColor(image image.Image) (r, g, b uint8) {
	temp := imaging.Resize(image, 20, 0, imaging.Lanczos)
	step := 1
	var count, sr, sg, sb uint32
	tempRect := temp.Bounds()
	for x := tempRect.Min.X; x < tempRect.Max.X; x += step {
		for y := tempRect.Min.Y; y < tempRect.Max.Y; y += step {
			r, g, b, _ := temp.At(x, y).RGBA()
			if r != 255 && g != 255 && b != 255 {
				sr += r & 0xFF
				sg += g & 0xFF
				sb += b & 0xFF
				count++
			}
		}
	}
	if count > 0 {
		return uint8(sr / count), uint8(sg / count), uint8(sb / count)
	}

	return 0, 0, 0

}

func ProminentColor(image image.Image) (r, g, b uint8) {
	//xresize := 40
	step := 4

	//temp := imaging.Resize(image, xresize, 0, imaging.Linear)
	colorsCount := make(map[uint32]int)
	tempRect := image.Bounds()
	currentMax := 0
	prominentColor := uint32(0)
	for x := tempRect.Min.X; x < tempRect.Max.X; x += step {
		for y := tempRect.Min.Y; y < tempRect.Max.Y; y += step {
			r, g, b, a := image.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			r = r >> 8
			g = g >> 8
			b = b >> 8
			if r == 0xFF && g == 0xFF && b == 0xFF {
				// ignore white pixels
				continue
			}
			color := ((r & 0xFE) << 16) | ((g & 0xFE) << 8) | (b & 0xFE)
			count, hasKey := colorsCount[color]
			if hasKey {
				colorsCount[color] = count + 1
				if colorsCount[color] > currentMax {
					prominentColor = color
					currentMax = colorsCount[color]
				}
			} else {
				colorsCount[color] = 1
			}
		}
	}
	return uint8(prominentColor >> 16 & 0xff), uint8(prominentColor >> 8 & 0xff), uint8(prominentColor & 0xff)
}
