package pixelpack

import (
	"github.com/dusk125/rectpack"
	"github.com/faiface/pixel"
)

type PixelPacker struct {
	internal rectpack.Packer
	batch    *pixel.Batch
	img      *pixel.PictureData
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
		irect  = pack.internal.Get(id)
		rect   = pixel.R(float64(irect.Min.X), float64(irect.Min.Y), float64(irect.Max.X), float64(irect.Max.Y))
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
