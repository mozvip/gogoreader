package crop

import (
	"image"
)

// comparator is a function that returns a difference between two colors in
// range 0.0..1.0 (0.0 - same colors, 1.0 - totally different colors).
type comparator func(r1, g1, b1, r2, g2, b2 uint32) float64

// CmpRGBComponents returns RGB components difference of two colors.
func CmpRGBComponents(r1, g1, b1, r2, g2, b2 uint32) float64 {
	const maxDiff = 765.0 // Difference between black and white colors

	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8

	return float64((max(r1, r2)-min(r1, r2))+
		(max(g1, g2)-min(g1, g2))+
		(max(b1, b2)-min(b1, b2))) / maxDiff
}

// min is minimum of two uint32
func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// max is maximum of two uint32
func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func CropBorders(img image.Image) image.Rectangle {
	return CropBordersWithComparator(img, CmpRGBComponents)
}

func avgColorForLine(img image.Image, y int) (r, g, b uint32) {
	var sumR, sumG, sumB uint32
	var c uint32
	rectangle := img.Bounds()
	for x := rectangle.Min.X; x < rectangle.Max.X; x++ {
		r, g, b, _ := img.At(x, y).RGBA()
		sumR += r
		sumG += g
		sumB += b
		c++
	}
	return sumR / c, sumG / c, sumB / c
}

func avgColorForColumn(img image.Image, x int) (r, g, b uint32) {
	var sumR, sumG, sumB uint32
	var c uint32
	rectangle := img.Bounds()
	for y := rectangle.Min.Y; y < rectangle.Max.Y; y++ {
		r, g, b, _ := img.At(x, y).RGBA()
		sumR += r
		sumG += g
		sumB += b
		c++
	}
	return sumR / c, sumG / c, sumB / c
}

func CropBordersWithComparator(img image.Image, comparator comparator) image.Rectangle {
	rectangle := img.Bounds()

	threshold := 0.10
	countThreshold := 0.02

	r, g, b := avgColorForLine(img, 0)
	maxBad := int(float64(rectangle.Max.X-rectangle.Min.X) * countThreshold)
	badCount := 0
TopLoop:
	for y := rectangle.Min.Y; y < rectangle.Max.Y; y++ {
		rectangle.Min.Y = y
		badCount = 0
		for x := rectangle.Min.X; x < rectangle.Max.X; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			if comparator(r1, g1, b1, r, g, b) > threshold {
				badCount++
				if badCount > maxBad {
					break TopLoop
				}
			}
		}
	}

	r, g, b = avgColorForLine(img, rectangle.Max.Y-1)
BottomLoop:
	for y := rectangle.Max.Y - 1; y >= rectangle.Min.Y; y-- {
		rectangle.Max.Y = y + 1
		badCount = 0
		for x := rectangle.Min.X; x < rectangle.Max.X; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			if comparator(r1, g1, b1, r, g, b) > threshold {
				badCount++
				if badCount > maxBad {
					break BottomLoop
				}
			}
		}
	}

	maxBad = int(float64(rectangle.Max.Y-rectangle.Min.Y) * countThreshold)
	r, g, b = avgColorForColumn(img, 0)
LeftLoop:
	for x := rectangle.Min.X; x < rectangle.Max.X; x++ {
		badCount = 0
		rectangle.Min.X = x
		for y := rectangle.Min.Y; y < rectangle.Max.Y; y++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			if comparator(r1, g1, b1, r, g, b) > threshold {
				badCount++
				if badCount > maxBad {
					break LeftLoop
				}
			}
		}

	}

	r, g, b = avgColorForColumn(img, rectangle.Max.X-1)
RightLoop:
	for x := rectangle.Max.X - 1; x >= rectangle.Min.X; x-- {
		badCount = 0
		rectangle.Max.X = x + 1
		for y := rectangle.Min.Y; y < rectangle.Max.Y; y++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			if comparator(r1, g1, b1, r, g, b) > threshold {
				badCount++
				if badCount > maxBad {
					break RightLoop
				}
			}
		}
	}

	return rectangle
}
