package gogoreader

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
	"github.com/faiface/pixel"
)

func AverageColor(image image.Image) (r, g, b uint8) {
	temp := imaging.Resize(image, 20, 0, imaging.NearestNeighbor)
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

func ProminentColor(pictureData *pixel.PictureData, rect image.Rectangle) pixel.RGBA {
	step := 3

	colorsCount := make(map[pixel.RGBA]uint)
	currentMax := uint(0)
	prominentColor := pixel.RGB(0, 0, 0)
	for x := rect.Min.X; x < rect.Max.X; x += step {
		for y := rect.Min.Y; y < rect.Max.Y; y += step {
			rgba := pictureData.Color(pixel.Vec{X: float64(x), Y: float64(y)})
			r, g, b := rgba.R, rgba.G, rgba.B
			if r > 0.95 && g > 0.95 && b > 0.95 {
				// ignore white pixels
				continue
			}
			if r < 0.05 && g < 0.05 && b < 0.05 {
				// ignore black pixels
				continue
			}
			// remove precision
			r = math.Round(r*10) / 10
			g = math.Round(g*10) / 10
			b = math.Round(b*10) / 10
			color := pixel.RGB(r, g, b)
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
