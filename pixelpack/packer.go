package pixelpack

import (
	"image"

	"github.com/dusk125/rectpack"
	"github.com/faiface/pixel"
)

func imgRectToPix(r image.Rectangle) pixel.Rect {
	return pixel.R(float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y))
}

type PixelPacker struct {
	internal *rectpack.Packer
	batch    *pixel.Batch
	img      *pixel.PictureData
}

func NewPacker(width, height int, flags rectpack.CreateFlags) (p *PixelPacker) {
	p = &PixelPacker{
		internal: rectpack.NewPacker(width, height, flags),
	}
	return
}

func (pack *PixelPacker) Pack(flags rectpack.PackFlags) (err error) {
	if err = pack.internal.Pack(flags); err != nil {
		return
	}
	pack.img = pixel.PictureDataFromImage(pack.internal.Image())
	pack.batch = pixel.NewBatch(&pixel.TrianglesData{}, pack.img)
	return
}

// Draws the given texture to the batch
func (pack *PixelPacker) Draw(id int, m pixel.Matrix) {
	var (
		rect   = imgRectToPix(pack.internal.Get(id))
		sprite = pixel.NewSprite(pack.img, rect)
	)

	sprite.Draw(pack.batch, m)
}

// Draws the internal batch to the given target
func (pack *PixelPacker) DrawTo(t pixel.Target) {
	pack.batch.Draw(t)
}

// Clear the internal batch of drawn sprites
func (pack *PixelPacker) Clear() {
	pack.batch.Clear()
}
