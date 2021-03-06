package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
)

type Score struct {
	index    int
	distance float64
}

func neighbourFor(index int, strips []image.Image) Score {

	min := Score{-1, math.Inf(1)}

	b1 := strips[index].Bounds()

	for i, s := range strips {
		if i == index {
			continue
		}

		b2 := s.Bounds()

		d := distance(strips[index], b1.Max.X-1, s, b2.Min.X)

		if d < min.distance {
			min = Score{i, d}
		}
	}

	return min
}

func distance(sl1 image.Image, col1 int, sl2 image.Image, col2 int) float64 {

	d := float64(0)

	b1 := sl1.Bounds()

	for y := b1.Min.Y; y < b1.Max.Y-8; y += 8 {

		r1, g1, b1, _ := sl1.At(col1, y).RGBA()
		r2, g2, b2, _ := sl2.At(col2, y).RGBA()

		dr := float64(int16(r1) - int16(r2))
		dg := float64(int16(g1) - int16(g2))
		db := float64(int16(b1) - int16(b2))

		d += math.Sqrt(dr*dr + db*db + dg*dg)
	}

	return d
}

func guessStripWidth(img image.Image) int {

	b := img.Bounds()

	distances := make([]float64, b.Dx())

	sum := float64(0)
	sumsq := float64(0)

	for x := b.Min.X; x < b.Max.X; x++ {
		d := distance(img, x, img, x+1)
		distances[x-b.Min.X] = d
		sum += d
		sumsq += d * d

	}

	n := float64(b.Dx())

	mean := sum / n
	stddev := math.Sqrt(sumsq/n - mean*mean)

	// we now have the mean and stddev of all the distances between sequential pixel columns
	// figure out the most common width between distances which are 2 stddevs from the mean

	votes := make(map[int]int)

	prev := -1
	for i := 1; i < len(distances); i++ {
		if distances[i] > distances[i-1]+2*stddev {
			fmt.Print(i-prev, " ")
			votes[i-prev]++
			prev = i
		}
	}
	fmt.Println()

	width := 0
	maxvotes := 0
	for k, v := range votes {
		if maxvotes < v {
			width = k
			maxvotes = v
		}
	}

	return width
}

func guessLeftmostNoLeftMatch(rightof []Score) int {

	seen := make([]bool, len(rightof))

	for _, r := range rightof {
		seen[r.index] = true
	}

	leftmost := -1
	notseen := 0
	for i := 0; i < len(rightof); i++ {
		if !seen[i] {
			leftmost = i
			notseen++
		}
	}

	if notseen == 1 {
		return leftmost
	}

	return -1

}

func guessLeftmostHighestRelativeError(strips []image.Image, rightof []Score) int {

	rightmost := -1
	ravg := float64(0)

	for i, r := range rightof {

		b := strips[i].Bounds()
		d0 := distance(strips[i], b.Max.X-3, strips[i], b.Max.X-2)
		d1 := distance(strips[i], b.Max.X-2, strips[i], b.Max.X-1)

		b = strips[r.index].Bounds()
		d2 := distance(strips[r.index], b.Min.X, strips[r.index], b.Min.X+1)
		d3 := distance(strips[r.index], b.Min.X+1, strips[r.index], b.Min.X+2)

		avg := (d0 + d1 + d2 + d3) / 4.0

		if rightmost == -1 || math.Abs(rightof[rightmost].distance-ravg)/ravg < math.Abs(r.distance-avg)/avg {
			rightmost = i
			ravg = avg
		}
	}

	return rightof[rightmost].index
}

// fisher-yates
func shuffle(array []image.Image) {

	for i := len(array) - 1; i >= 1; i-- {
		j := rand.Intn(i + 1)
		array[i], array[j] = array[j], array[i]
	}
}

func shredImage(input, output string, stripwidth int) {

	r, err := os.Open(input)
	if err != nil {
		fmt.Println("error during open: ", err)
		return
	}

	decoded, _, err := image.Decode(r)
	if err != nil {
		fmt.Println("error during decode: ", err)
		return
	}

	img := decoded.(*image.YCbCr)

	fmt.Println("image is: ", img.Bounds())

	nstrip := img.Bounds().Dx() / stripwidth

	strips := make([]image.Image, nstrip)

	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := img.Bounds().Dy()
		strips[i] = img.SubImage(image.Rect(x0, y0, x1, y1))
	}

	shuffle(strips)

	shredded := image.NewNRGBA(img.Bounds())

	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := img.Bounds().Dy()
		draw.Draw(shredded, image.Rect(x0, y0, x1, y1), strips[i], strips[i].Bounds().Min, draw.Src)
	}

	fmt.Println("encoding to ", output)

	po, _ := os.Create(output)
	png.Encode(po, shredded)
	po.Close()
}

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func splitImage(nstrip, stripwidth, dy int, si SubImager) []image.Image {

	strips := make([]image.Image, nstrip)

	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := dy
		strips[i] = si.SubImage(image.Rect(x0, y0, x1, y1))
	}

	return strips
}

func main() {

	var optStripWidth = flag.Int("stripwidth", 0, "the width of the image strips")
	var optShred = flag.Bool("shred", false, "shred image")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Println("usage: input output.jpg")
		os.Exit(1)
	}

	input_filename := flag.Arg(0)
	fmt.Println("input file: ", input_filename)
	output_filename := flag.Arg(1)

	if *optShred {
		shredImage(input_filename, output_filename, *optStripWidth)
		return
	}

	r, _ := os.Open(input_filename)
	img, _, _ := image.Decode(r)

	fmt.Println("image is: ", img.Bounds())

	var stripwidth int

	if *optStripWidth == 0 {
		stripwidth = guessStripWidth(img)
	} else {
		stripwidth = *optStripWidth
	}

	fmt.Println("using ", stripwidth, " as strip width")

	nstrip := img.Bounds().Dx() / stripwidth

	var strips []image.Image

	switch t := img.(type) {
	case *image.NRGBA:
		strips = splitImage(nstrip, stripwidth, img.Bounds().Dy(), t)
	case *image.RGBA:
		strips = splitImage(nstrip, stripwidth, img.Bounds().Dy(), t)
	case *image.YCbCr:
		strips = splitImage(nstrip, stripwidth, img.Bounds().Dy(), t)
	}

	rightof := make([]Score, nstrip)

	for i := 0; i < nstrip; i++ {
		rightof[i] = neighbourFor(i, strips)
		fmt.Println("right neighbour for ", i, " = ", rightof[i])
	}

	leftmost := guessLeftmostNoLeftMatch(rightof)
	if leftmost == -1 {
		leftmost = guessLeftmostHighestRelativeError(strips, rightof)
	}

	fmt.Println("using strip", leftmost, "as leftmost")

	unshredded := image.NewNRGBA(img.Bounds())

	n := leftmost
	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := img.Bounds().Dy()
		fmt.Print(" ", n)
		draw.Draw(unshredded, image.Rect(x0, y0, x1, y1), strips[n], strips[n].Bounds().Min, draw.Src)
		n = rightof[n].index
	}

	fmt.Println()

	po, _ := os.Create(output_filename)
	jpeg.Encode(po, unshredded, nil)
	po.Close()

	fmt.Println("unshredded image written to: ", output_filename)
}
