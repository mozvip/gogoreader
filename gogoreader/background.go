package gogoreader

import (
	"image"
	"math"

	"github.com/faiface/pixel"
)

func AverageColor(pictureData *pixel.PictureData, rect image.Rectangle) pixel.RGBA {
	step := 3.0
	var count, sr, sg, sb float64
	for x := float64(rect.Min.X); x < float64(rect.Max.X); x += step {
		for y := float64(rect.Min.Y); y < float64(rect.Max.Y); y += step {
			rgba := pictureData.Color(pixel.Vec{X: x, Y: y})
			r, g, b := rgba.R, rgba.G, rgba.B
			if r > 0.95 && g > 0.95 && b > 0.95 {
				// ignore white pixels
				continue
			}
			sr += r
			sg += g
			sb += b
			count++
		}
	}
	if count > 0 {
		return pixel.RGB(sr/count, sg/count, sb/count)
	}

	return pixel.RGB(0, 0, 0)

}

func ProminentColor(pictureData *pixel.PictureData, rect image.Rectangle) pixel.RGBA {
	step := 3.0

	colorsCount := make(map[pixel.RGBA]uint)
	currentMax := uint(0)
	prominentColor := pixel.RGB(0, 0, 0)
	for x := float64(rect.Min.X); x < float64(rect.Max.X); x += step {
		for y := float64(rect.Min.Y); y < float64(rect.Max.Y); y += step {
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
