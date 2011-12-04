package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"image/ycbcr"
	"math"
	"math/rand"
	"os"

	_ "image/jpeg"
)

type Score struct {
	index    int
	distance uint64
}

func neighbourFor(index int, strips []image.Image) Score {

	min := Score{-1, uint64(1 << 63)}

	for i, s := range strips {
		if i == index {
			continue
		}

		b1 := strips[index].Bounds()
		b2 := s.Bounds()

		d := distance(strips[index], b1.Max.X-1, s, b2.Min.X)

		if d < min.distance {
			min = Score{i, d}
		}
	}

	return min
}

func rgb2yuv(c color.NRGBA) (y, u, v float32) {

	R := float32(c.R) / float32(255)
	G := float32(c.G) / float32(255)
	B := float32(c.B) / float32(255)

	Y := 0.299*R + 0.587*G + 0.114*B
	U := -0.14713*R - 0.28886*G + 0.436*B
	V := 0.615*R - 0.51499*G - 0.10001*B

	return Y, U, V

}

func distance(sl1 image.Image, col1 int, sl2 image.Image, col2 int) uint64 {

	d := uint64(0)

	b1 := sl1.Bounds()

	for y := b1.Min.Y; y < b1.Max.Y; y++ {

		c1 := sl1.At(col1, y).(color.NRGBA)
		c2 := sl2.At(col2, y).(color.NRGBA)

		dr := float64(int16(c1.R) - int16(c2.R))
		dg := float64(int16(c1.G) - int16(c2.G))
		db := float64(int16(c1.B) - int16(c2.B))

		d += uint64(math.Sqrt(dr*dr + db*db + dg*dg))
	}

	return d
}

func guessStripWidth(img image.Image) int {
	// placeholder 
	return 32
}

func guessLeftMostNoLeftMatch(rightof []Score) int {

	seen := make([]bool, len(rightof), len(rightof))

	for i := 0; i < len(rightof); i++ {
		seen[i] = false
	}

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

func guessLeftMostHighestAverageError(strips []image.Image, rightof []Score) int {

	rightmost := -1
	ravg := float64(0)

	for i, r := range rightof {

		b := strips[i].Bounds()
		d0 := distance(strips[i], b.Max.X-3, strips[i], b.Max.X-2)
		d1 := distance(strips[i], b.Max.X-2, strips[i], b.Max.X-1)

		b = strips[r.index].Bounds()
		d2 := distance(strips[r.index], b.Min.X, strips[r.index], b.Min.X+1)
		d3 := distance(strips[r.index], b.Min.X+1, strips[r.index], b.Min.X+2)

		avg := float64(d0+d1+d2+d3) / 4.0

		if rightmost == -1 || math.Abs(float64(rightof[rightmost].distance)-ravg)/ravg < math.Abs(float64(r.distance)-avg)/avg {
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

	img := decoded.(*ycbcr.YCbCr)

	fmt.Println("image is: ", img.Bounds())

	nstrip := img.Bounds().Dx() / stripwidth

	strips := make([]image.Image, nstrip, nstrip)

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

func main() {

	var optStripWidth = flag.Int("stripwidth", 32, "the width of the image strips")
	var optShred = flag.Bool("shred", false, "shred image")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Println("usage: input.png output.png")
		os.Exit(1)
	}

	input_filename := flag.Arg(0)
	fmt.Println("input file: ", input_filename)
	output_filename := flag.Arg(1)

	if *optShred {
		shredImage(input_filename, output_filename, *optStripWidth)
		os.Exit(1)
	}

	r, _ := os.Open(input_filename)
	pngimg, _ := png.Decode(r)

	img := pngimg.(*image.NRGBA)

	fmt.Println("image is: ", img.Bounds())

	var stripwidth int

	if *optStripWidth == 0 {
		stripwidth = guessStripWidth(img)
	} else {
		stripwidth = *optStripWidth
	}

	nstrip := img.Bounds().Dx() / stripwidth

	strips := make([]image.Image, nstrip, nstrip)

	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := img.Bounds().Dy()
		strips[i] = img.SubImage(image.Rect(x0, y0, x1, y1))
	}

	rightof := make([]Score, nstrip, nstrip)

	for i := 0; i < nstrip; i++ {
		rightof[i] = neighbourFor(i, strips)
		fmt.Println("right neighbour for ", i, " = ", rightof[i])
	}

	leftmost := guessLeftMostNoLeftMatch(rightof)
	if leftmost == -1 {
		leftmost = guessLeftMostHighestAverageError(strips, rightof)
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
	png.Encode(po, unshredded)
	po.Close()

	fmt.Println("unshredded image written to: ", output_filename)
}
