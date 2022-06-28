package crop

import (
	"image"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/mozvip/gomics/gogoreader"
)

// comparator is a function that returns a difference between two colors in
// range 0..255 (0 - same colors, 255 - totally different colors).
type comparator func(r1, g1, b1, r2, g2, b2 uint8) uint32

// CmpRGBComponents returns RGB components difference of two colors.
func CmpRGBComponents(r1, g1, b1, r2, g2, b2 uint8) uint32 {
	return uint32((max(r1, r2)-min(r1, r2))+
		(max(g1, g2)-min(g1, g2))+
		(max(b1, b2)-min(b1, b2))) / 3
}

func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func max(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

func CropBorders(img *image.NRGBA, rect *rl.Rectangle) {
	CropBordersWithComparator(img, rect, CmpRGBComponents)
}

func avgColorForLine(img *image.NRGBA, y uint32, minX uint32, maxX uint32) (r, g, b uint8) {
	var sumR, sumG, sumB uint32
	var c uint32
	for x := minX; x < maxX; x++ {
		r, g, b := gogoreader.GetPixelColor(img, x, y)
		sumR += uint32(r)
		sumG += uint32(g)
		sumB += uint32(b)
		c++
	}
	if c == 0 {
		return 0, 0, 0
	}
	return uint8(sumR / c), uint8(sumG / c), uint8(sumB / c)
}

func avgColorForColumn(img *image.NRGBA, x, minY, height uint32) (r, g, b uint8) {
	var sumR, sumG, sumB uint32
	var c uint32
	for y := minY; y < minY+height; y++ {
		r, g, b := gogoreader.GetPixelColor(img, x, y)
		sumR += uint32(r)
		sumG += uint32(g)
		sumB += uint32(b)
		c++
	}
	if c == 0 {
		return 0, 0, 0
	}
	return uint8(sumR / c), uint8(sumG / c), uint8(sumB / c)
}

func CropBordersWithComparator(img *image.NRGBA, rect *rl.Rectangle, comparator comparator) {
	threshold := uint32(10)
	countThreshold := float32(0.01)

	var wg sync.WaitGroup

	rectMinY := uint32(rect.Y)
	rectMinX := uint32(rect.X)
	rectWidth := uint32(rect.Width)
	rectHeight := uint32(rect.Height)

	const step = 4

	maxBadForX := int(rect.Width * countThreshold)
	wg.Add(1)
	// identify top line : rectMinY
	go func() {
		defer wg.Done()
		r, g, b := avgColorForLine(img, rectMinY, rectMinX, rectMinX+rectWidth)
		for minY := rectMinY; minY < rectMinY+rectHeight; minY++ {
			badCount := 0
			var x uint32
			for x = rectMinX; x < rectMinX+rectWidth; x += step {
				r1, g1, b1 := gogoreader.GetPixelColor(img, x, minY)
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			rectMinY++
		}
	}()

	wg.Add(1)
	// identify bottom line
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, uint32(rect.Y)+uint32(rect.Height)-1, rectMinX, rectMinX+uint32(rect.Width)-1)
		for height := rectHeight - 1; height > 0; height-- {
			badCount := 0
			y := uint32(rectMinY + height - 1)
			for x := rectMinX; x < rectMinX+rectWidth; x += step {
				r1, g1, b1 := gogoreader.GetPixelColor(img, x, y)
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			rectHeight--
		}
	}()

	wg.Wait()

	maxBadForY := int(rect.Height * countThreshold)
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, rectMinX, rectMinY, rectHeight)
		var minX uint32
		for minX = rectMinX; minX < rectMinX+rectWidth; minX++ {
			badCount := 0
			for y := rect.Y; y < rect.Y+rect.Height; y += step {
				r1, g1, b1 := gogoreader.GetPixelColor(img, uint32(minX), uint32(y))
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			rectMinX++
		}
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, rectMinX+rectWidth-1, rectMinY, rectHeight)
		for maxX := rectMinX + rectWidth - 1; maxX > rectMinX; maxX-- {
			badCount := 0
			for y := rectMinY; y < rectMinY+rectHeight; y += step {
				r1, g1, b1 := gogoreader.GetPixelColor(img, maxX, y)
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			rectWidth--
		}
	}()

	wg.Wait()

	rect.X = float32(rectMinX)
	rect.Width = float32(rectWidth)
	rect.Y = float32(rectMinY)
	rect.Height = float32(rectHeight)
}
