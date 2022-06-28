package crop

import (
	"image"
	"sync"
	"sync/atomic"

	"github.com/faiface/pixel"
)

// comparator is a function that returns a difference between two colors in
// range 0.0..1.0 (0.0 - same colors, 1.0 - totally different colors).
type comparator func(r1, g1, b1, r2, g2, b2 float64) float64

// CmpRGBComponents returns RGB components difference of two colors.
func CmpRGBComponents(r1, g1, b1, r2, g2, b2 float64) float64 {
	const maxDiff = 765.0 // Difference between black and white colors

	return (max(r1, r2) - min(r1, r2)) +
		(max(g1, g2) - min(g1, g2)) +
		(max(b1, b2)-min(b1, b2))/maxDiff
}

// min is minimum of two uint32
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max is maximum of two uint32
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func CropBorders(img *pixel.PictureData, rect *image.Rectangle) {
	CropBordersWithComparator(img, rect, CmpRGBComponents)
}

func avgColorForLine(img *pixel.PictureData, y, minX, maxX int32) (r, g, b float64) {
	var sumR, sumG, sumB float64
	var c float64
	for x := minX; x < maxX; x++ {
		color := img.Color(pixel.V(float64(x), float64(y)))
		r, g, b := color.R, color.G, color.B
		sumR += r
		sumG += g
		sumB += b
		c++
	}
	if c == 0 {
		return 0, 0, 0
	}
	return sumR / c, sumG / c, sumB / c
}

func avgColorForColumn(img *pixel.PictureData, x, minY, maxY int32) (r, g, b float64) {
	var sumR, sumG, sumB float64
	var c float64
	for y := minY; y < maxY; y++ {
		color := img.Color(pixel.V(float64(x), float64(y)))
		r, g, b := color.R, color.G, color.B
		sumR += r
		sumG += g
		sumB += b
		c++
	}
	if c == 0 {
		return 0, 0, 0
	}
	return sumR / c, sumG / c, sumB / c
}

func CropBordersWithComparator(img *pixel.PictureData, rect *image.Rectangle, comparator comparator) {
	threshold := 0.10
	countThreshold := 0.01

	var wg sync.WaitGroup

	rectMinY := int32(rect.Min.Y)
	rectMaxY := int32(rect.Max.Y)
	rectMinX := int32(rect.Min.X)
	rectMaxX := int32(rect.Max.X)

	const step = 4

	maxBadForX := int(float64(rect.Dx()) * countThreshold)
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, atomic.LoadInt32(&rectMinY), atomic.LoadInt32(&rectMinX), atomic.LoadInt32(&rectMaxX))
		for minY := atomic.LoadInt32(&rectMinY); minY < atomic.LoadInt32(&rectMaxY); minY++ {
			badCount := 0
			for x := atomic.LoadInt32(&rectMinX); x < atomic.LoadInt32(&rectMaxX); x += step {
				color := img.Color(pixel.V(float64(x), float64(minY)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			atomic.AddInt32(&rectMinY, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, atomic.LoadInt32(&rectMaxY)-1, atomic.LoadInt32(&rectMinX), atomic.LoadInt32(&rectMaxX)-1)
		for maxY := atomic.LoadInt32(&rectMaxY) - 1; maxY > atomic.LoadInt32(&rectMinY); maxY-- {
			badCount := 0
			for x := atomic.LoadInt32(&rectMinX); x < atomic.LoadInt32(&rectMaxX); x += step {
				color := img.Color(pixel.V(float64(x), float64(maxY)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			atomic.AddInt32(&rectMaxY, -1)
		}
	}()

	maxBadForY := int(float32(rect.Max.Y-rect.Min.Y) * float32(countThreshold))
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, atomic.LoadInt32(&rectMinX), atomic.LoadInt32(&rectMinY), atomic.LoadInt32(&rectMaxY))
		for minX := int(atomic.LoadInt32(&rectMinX)); minX < int(atomic.LoadInt32(&rectMaxX)); minX++ {
			badCount := 0
			for y := rect.Min.Y; y < rect.Max.Y; y += step {
				color := img.Color(pixel.V(float64(minX), float64(y)))
				r1, g1, b1 := color.R, color.G, color.B

				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			atomic.AddInt32(&rectMinX, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, atomic.LoadInt32(&rectMaxX)-1, atomic.LoadInt32(&rectMinY), atomic.LoadInt32(&rectMaxY)-1)
		for maxX := atomic.LoadInt32(&rectMaxX) - 1; maxX > atomic.LoadInt32(&rectMinX); maxX-- {
			badCount := 0
			for y := atomic.LoadInt32(&rectMinY); y < atomic.LoadInt32(&rectMaxY); y += step {
				color := img.Color(pixel.V(float64(maxX), float64(y)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			atomic.AddInt32(&rectMaxX, -1)
		}
	}()

	wg.Wait()

	rect.Min.X = int(rectMinX)
	rect.Max.X = int(rectMaxX)
	rect.Min.Y = int(rectMinY)
	rect.Max.Y = int(rectMaxY)
}
