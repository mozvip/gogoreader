package crop

import (
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

func CropBorders(img *pixel.PictureData, rect *pixel.Rect) {
	CropBordersWithComparator(img, rect, CmpRGBComponents)
}

func avgColorForLine(img *pixel.PictureData, y float64, minX float64, maxX float64) (r, g, b float64) {
	var sumR, sumG, sumB float64
	var c float64
	for x := minX; x < maxX; x++ {
		color := img.Color(pixel.V(x, y))
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

func avgColorForColumn(img *pixel.PictureData, x, minY, maxY float64) (r, g, b float64) {
	var sumR, sumG, sumB float64
	var c float64
	for y := minY; y < maxY; y++ {
		color := img.Color(pixel.V(x, y))
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

func CropBordersWithComparator(img *pixel.PictureData, rect *pixel.Rect, comparator comparator) {
	threshold := 0.10
	countThreshold := 0.01

	var wg sync.WaitGroup

	rectMinY := int64(rect.Min.Y)
	rectMaxY := int64(rect.Max.Y)
	rectMinX := int64(rect.Min.X)
	rectMaxX := int64(rect.Max.X)

	const step = 4

	maxBadForX := rect.W() * countThreshold
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, float64(atomic.LoadInt64(&rectMinY)), float64(atomic.LoadInt64(&rectMinX)), float64(atomic.LoadInt64(&rectMaxX)))
		for minY := atomic.LoadInt64(&rectMinY); minY < atomic.LoadInt64(&rectMaxY); minY++ {
			badCount := 0.0
			for x := atomic.LoadInt64(&rectMinX); x < atomic.LoadInt64(&rectMaxX); x += step {
				color := img.Color(pixel.V(float64(x), float64(minY)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			atomic.AddInt64(&rectMinY, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, float64(atomic.LoadInt64(&rectMaxY)-1), float64(atomic.LoadInt64(&rectMinX)), float64(atomic.LoadInt64(&rectMaxX)-1))
		for maxY := atomic.LoadInt64(&rectMaxY) - 1; maxY > atomic.LoadInt64(&rectMinY); maxY-- {
			badCount := 0.0
			for x := atomic.LoadInt64(&rectMinX); x < atomic.LoadInt64(&rectMaxX); x += step {
				color := img.Color(pixel.V(float64(x), float64(maxY)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForX {
						return
					}
				}
			}
			atomic.AddInt64(&rectMaxY, -1)
		}
	}()

	maxBadForY := (rect.Max.Y - rect.Min.Y) * countThreshold
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, float64(atomic.LoadInt64(&rectMinX)), float64(atomic.LoadInt64(&rectMinY)), float64(atomic.LoadInt64(&rectMaxY)))
		for minX := int(atomic.LoadInt64(&rectMinX)); minX < int(atomic.LoadInt64(&rectMaxX)); minX++ {
			badCount := 0.0
			for y := rect.Min.Y; y < rect.Max.Y; y += step {
				color := img.Color(pixel.V(float64(minX), y))
				r1, g1, b1 := color.R, color.G, color.B

				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			atomic.AddInt64(&rectMinX, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, float64(atomic.LoadInt64(&rectMaxX)-1), float64(atomic.LoadInt64(&rectMinY)), float64(atomic.LoadInt64(&rectMaxY)-1))
		for maxX := atomic.LoadInt64(&rectMaxX) - 1; maxX > atomic.LoadInt64(&rectMinX); maxX-- {
			badCount := 0.0
			for y := atomic.LoadInt64(&rectMinY); y < atomic.LoadInt64(&rectMaxY); y += step {
				color := img.Color(pixel.V(float64(maxX), float64(y)))
				r1, g1, b1 := color.R, color.G, color.B
				if comparator(r1, g1, b1, r, g, b) > threshold {
					badCount++
					if badCount > maxBadForY {
						return
					}
				}
			}
			atomic.AddInt64(&rectMaxX, -1)
		}
	}()

	wg.Wait()

	rect.Min.X = float64(rectMinX)
	rect.Max.X = float64(rectMaxX)
	rect.Min.Y = float64(rectMinY)
	rect.Max.Y = float64(rectMaxY)
}
