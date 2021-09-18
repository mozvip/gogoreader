package crop

import (
	"image"
	"sync"
	"sync/atomic"
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

func CropBorders(img image.Image, rect *image.Rectangle) {
	CropBordersWithComparator(img, rect, CmpRGBComponents)
}

func avgColorForLine(img image.Image, y int64, minX int64, maxX int64) (r, g, b uint32) {
	var sumR, sumG, sumB uint32
	var c uint32
	for x := minX; x < maxX; x++ {
		r, g, b, _ := img.At(int(x), int(y)).RGBA()
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

func avgColorForColumn(img image.Image, x, minY, maxY int64) (r, g, b uint32) {
	var sumR, sumG, sumB uint32
	var c uint32
	for y := minY; y < maxY; y++ {
		r, g, b, _ := img.At(int(x), int(y)).RGBA()
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

func CropBordersWithComparator(img image.Image, rect *image.Rectangle, comparator comparator) {
	threshold := 0.10
	countThreshold := 0.02

	var wg sync.WaitGroup

	rectMinY := int64(rect.Min.Y)
	rectMaxY := int64(rect.Max.Y)
	rectMinX := int64(rect.Min.X)
	rectMaxX := int64(rect.Max.X)

	const precision = 4

	maxBadForX := int(float64(rect.Max.X-rect.Min.X) * countThreshold)
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForLine(img, atomic.LoadInt64(&rectMinY), atomic.LoadInt64(&rectMinX), atomic.LoadInt64(&rectMaxX))
		for minY := atomic.LoadInt64(&rectMinY); minY < atomic.LoadInt64(&rectMaxY); minY++ {
			badCount := 0
			for x := atomic.LoadInt64(&rectMinX); x < atomic.LoadInt64(&rectMaxX); x += precision {
				r1, g1, b1, _ := img.At(int(x), int(minY)).RGBA()
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

		r, g, b := avgColorForLine(img, atomic.LoadInt64(&rectMaxY)-1, atomic.LoadInt64(&rectMinX), atomic.LoadInt64(&rectMaxX)-1)
		for maxY := atomic.LoadInt64(&rectMaxY) - 1; maxY > atomic.LoadInt64(&rectMinY); maxY-- {
			badCount := 0
			for x := atomic.LoadInt64(&rectMinX); x < atomic.LoadInt64(&rectMaxX); x += precision {
				r1, g1, b1, _ := img.At(int(x), int(maxY)).RGBA()
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

	maxBadForY := int(float64(rect.Max.Y-rect.Min.Y) * countThreshold)
	wg.Add(1)
	go func() {
		defer wg.Done()

		r, g, b := avgColorForColumn(img, atomic.LoadInt64(&rectMinX), atomic.LoadInt64(&rectMinY), atomic.LoadInt64(&rectMaxY))
		for minX := int(atomic.LoadInt64(&rectMinX)); minX < int(atomic.LoadInt64(&rectMaxX)); minX++ {
			badCount := 0
			for y := rect.Min.Y; y < rect.Max.Y; y += precision {
				r1, g1, b1, _ := img.At(minX, y).RGBA()
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

		r, g, b := avgColorForColumn(img, atomic.LoadInt64(&rectMaxX)-1, atomic.LoadInt64(&rectMinY), atomic.LoadInt64(&rectMaxY)-1)
		for maxX := atomic.LoadInt64(&rectMaxX) - 1; maxX > atomic.LoadInt64(&rectMinX); maxX-- {
			badCount := 0
			for y := atomic.LoadInt64(&rectMinY); y < atomic.LoadInt64(&rectMaxY); y += precision {
				r1, g1, b1, _ := img.At(int(maxX), int(y)).RGBA()
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

	rect.Min.X = int(rectMinX)
	rect.Max.X = int(rectMaxX)
	rect.Min.Y = int(rectMinY)
	rect.Max.Y = int(rectMaxY)
}
