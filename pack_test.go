package rectpack_test

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/dusk125/rectpack"
	"golang.org/x/image/colornames"
)

func fill(w, h int, c color.Color) (img *image.RGBA) {
	img = image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	return
}

func colorEq(i2 image.Image, w, h int, c color.Color) (err error) {
	var i1 = fill(w, h, c)

	if !i1.Bounds().Size().Eq(i2.Bounds().Size()) {
		return fmt.Errorf("Image sizes are not the same: Expected: %s, Got: %s", i1.Bounds().Size(), i2.Bounds().Size())
	}

	for x := 0; x < i1.Bounds().Dx(); x++ {
		for y := 0; y < i1.Bounds().Dy(); y++ {
			r1, g1, b1, a1 := i1.At(x, y).RGBA()
			r2, g2, b2, a2 := i2.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return fmt.Errorf("At: (%d, %d), Expected: (%v, %v, %v, %v), Got: (%v, %v, %v, %v)", x, y, r1, g1, b1, a1, r2, b2, g2, a2)
			}
		}
	}

	return nil
}

func TestNewPacker(t *testing.T) {
	t.Run("Test", func(t *testing.T) {
		pack := rectpack.NewPacker(rectpack.PackerCfg{})
		colors := []struct {
			col  color.Color
			w, h int
		}{
			{
				col: colornames.Black,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Aliceblue,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Navy,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Salmon,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Orchid,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Olive,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
			{
				col: colornames.Oldlace,
				w:   rand.Intn(1024),
				h:   rand.Intn(1024),
			},
		}
		for i, c := range colors {
			pack.Insert(i, fill(c.w, c.h, c.col))
		}
		if err := pack.Pack(); err != nil {
			t.Error(err)
		}
		if err := pack.Save("test.png"); err != nil {
			t.Error(err)
		}
		for i, c := range colors {
			img := pack.SubImage(i)
			if err := colorEq(img, c.w, c.h, c.col); err != nil {
				t.Errorf("%d is not expected: %s", i, err.Error())
			}
		}
	})
}

func Save(filename string, img image.Image) (err error) {
	var (
		file *os.File
	)

	if err = os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}

	if file, err = os.Create(filename); err != nil {
		return
	}
	defer file.Close()

	switch path.Ext(filename) {
	case ".png":
		err = png.Encode(file, img)
	case ".jpeg", ".jpg":
		err = jpeg.Encode(file, img, nil)
	default:
		err = errors.New("Bad extension")
	}
	if err != nil {
		return
	}

	return
}
