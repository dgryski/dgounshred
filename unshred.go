package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

func saveStrips(strips []image.Image) {
	for i, s := range strips {
		w, _ := os.Create(fmt.Sprintf("Strip_%02d.png", i))
		png.Encode(w, s)
		w.Close()
	}
}

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

		d := distance(strips[index], s)

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

func distance(sl1, sl2 image.Image) uint64 {

	d := uint64(0)

	b1 := sl1.Bounds()
	b2 := sl2.Bounds()

	for y := b1.Min.Y; y < b1.Max.Y; y++ {

		c1 := sl1.At(b1.Max.X-1, y).(color.NRGBA)
		c2 := sl2.At(b2.Min.X, y).(color.NRGBA)

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

func guessLeftMost(rightof []Score) int {

        // Guess the left-most by assuming our matching algorithm places it
        // left of the rightmost slice (since every other slice will have a
        // better, actual match).  Not a terrible heuristic, bu fails on the
        // Tokyo test image due to higher internal mismatches (thanks to the
        // stupid black and white skyscraper)

	rightmost := 0

	for i, r := range rightof {
		if rightof[rightmost].distance < r.distance {
			rightmost = i
		}
	}

	return rightof[rightmost].index
}

func main() {

	var optStripWidth = flag.Int("stripwidth", 32, "the width of the image strips")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Println("usage: input.png output.png")
		os.Exit(1)
	}

	input_filename := flag.Arg(0)
	fmt.Println("input file: ", input_filename)
	output_filename := flag.Arg(1)

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

	leftmost := guessLeftMost(rightof[:])

	fmt.Println("using slice", leftmost, "as leftmost")

	unshredded := image.NewNRGBA(img.Bounds())

	n := leftmost
	for i := 0; i < nstrip; i++ {
		x0 := i * stripwidth
		y0 := 0
		x1 := x0 + stripwidth
		y1 := img.Bounds().Dy()
		fmt.Println("slice ", n)
		draw.Draw(unshredded, image.Rect(x0, y0, x1, y1), strips[n], strips[n].Bounds().Min, draw.Src)
		n = rightof[n].index
	}

	po, _ := os.Create(output_filename)
	png.Encode(po, unshredded)
	po.Close()

	fmt.Println("unshredded image written to: ", output_filename)
}
