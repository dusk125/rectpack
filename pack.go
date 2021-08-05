package rectpack

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"sort"
)

// This texture packer algorithm is based on this project
// https://github.com/TeamHypersomnia/rectpack2D

var (
	ErrorNoEmptySpace     = errors.New("Couldn't find an empty space")
	ErrorSplitFailed      = errors.New("Split failed")
	ErrGrowthFailed       = errors.New("A previously added texture failed to be added after packer growth")
	ErrUnsupportedSaveExt = errors.New("Unsupported save filename extension")
)

type PackFlags uint8
type CreateFlags uint8

const (
	AllowGrowth CreateFlags = 1 << iota // Should the packer space try to grow larger to fit oversized images
)

const (
	InsertFlipped PackFlags = 1 << iota // Should the sprite be inserted into the packer upside-down
)

type IPack interface {
	Pack(flags PackFlags) (err error)
}

type Packer struct {
	bounds      image.Rectangle
	emptySpaces []image.Rectangle
	queued      []queuedData
	rects       map[int]image.Rectangle
	images      map[int]*image.RGBA
	flags       CreateFlags
	pic         *image.RGBA
	nfId        int
}

// Creates a new packer instance
func NewPacker(width, height int, flags CreateFlags) (pack *Packer) {
	bounds := rect(0, 0, width, height)
	pack = &Packer{
		bounds:      bounds,
		flags:       flags,
		emptySpaces: []image.Rectangle{bounds},
		rects:       make(map[int]image.Rectangle),
		images:      make(map[int]*image.RGBA),
		queued:      make([]queuedData, 0),
	}
	return
}

// Inserts PictureData into the packer
func (pack *Packer) Insert(id int, pic *image.RGBA) (err error) {
	pack.queued = append(pack.queued, queuedData{id: id, pic: pic})
	return
}

// Helper to find the smallest empty space that'll fit the given bounds
func (pack Packer) find(bounds image.Rectangle) (index int, found bool) {
	for i, space := range pack.emptySpaces {
		if bounds.Dx() <= space.Dx() && bounds.Dy() <= space.Dy() {
			return i, true
		}
	}
	return
}

// Helper to remove a canidate empty space and return it
func (pack *Packer) remove(i int) (removed image.Rectangle) {
	removed = pack.emptySpaces[i]
	pack.emptySpaces = append(pack.emptySpaces[:i], pack.emptySpaces[i+1:]...)
	return
}

// Helper to increase the size of the internal texture and readd the queued textures to keep it defragmented
func (pack *Packer) grow(growBy image.Point, endex int) (err error) {
	newSize := pack.bounds.Size().Add(growBy)
	pack.bounds = rect(pack.bounds.Min.X, pack.bounds.Min.Y, newSize.X, newSize.Y)
	pack.emptySpaces = []image.Rectangle{pack.bounds}

	for _, data := range pack.queued[0:endex] {
		if err = pack.insert(data); err != nil {
			return
		}
	}

	return
}

// Helper to segment a found space so that the given data can fit in what's left
func (pack *Packer) insert(data queuedData) (err error) {
	var (
		s            *createdSplits
		bounds       = data.pic.Bounds()
		index, found = pack.find(bounds)
	)

	if !found {
		return ErrGrowthFailed
	}

	space := pack.remove(index)
	if s, err = split(bounds, space); err != nil {
		return
	}

	if s.hasBig {
		pack.emptySpaces = append(pack.emptySpaces, s.bigger)
	}
	if s.hasSmall {
		pack.emptySpaces = append(pack.emptySpaces, s.smaller)
	}

	sort.Slice(pack.emptySpaces, func(i, j int) bool {
		return area(pack.emptySpaces[i]) < area(pack.emptySpaces[j])
	})

	pack.rects[data.id] = rect(space.Min.X, space.Min.Y, bounds.Dx(), bounds.Dy())
	pack.images[data.id] = data.pic
	return
}

// Pack takes the added textures and packs them into the packer texture, growing the texture if necessary.
func (pack *Packer) Pack(flags PackFlags) (err error) {
	sort.Slice(pack.queued, func(i, j int) bool {
		return area(pack.queued[i].pic.Bounds()) > area(pack.queued[j].pic.Bounds())
	})

	for i, data := range pack.queued {
		var (
			bounds   = data.pic.Bounds()
			_, found = pack.find(bounds)
		)

		if !found {
			if pack.flags&AllowGrowth == 0 {
				return ErrorNoEmptySpace
			}

			if err = pack.grow(bounds.Size(), i); err != nil {
				return
			}
		}

		if err = pack.insert(data); err != nil {
			return
		}
	}

	pack.pic = image.NewRGBA(pack.bounds)
	for id, pic := range pack.images {
		for x := 0; x < pic.Bounds().Dx(); x++ {
			for y := 0; y < pic.Bounds().Dy(); y++ {
				rect := pack.rects[id]
				dstI := pack.pic.PixOffset(x+rect.Min.X, y+rect.Min.Y)
				var srcI int
				if flags&InsertFlipped != 0 {
					srcI = pic.PixOffset(x, y)
				} else {
					srcI = pic.PixOffset(x, y)
				}

				pack.pic.Pix[dstI] = pic.Pix[srcI]
			}
		}
	}
	pack.queued = pack.queued[:0]
	pack.emptySpaces = pack.emptySpaces[:0]
	pack.images = nil

	return
}

// Saves the internal texture as a file on disk, the output type is defined by the filename extension
func (pack *Packer) Save(filename string) (err error) {
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
		err = png.Encode(file, pack.pic)
	case ".jpeg", ".jpg":
		err = jpeg.Encode(file, pack.pic, nil)
	default:
		err = ErrUnsupportedSaveExt
	}
	if err != nil {
		return
	}

	return
}

func (pack *Packer) SetNotFoundId(id int) {
	pack.nfId = id
}

func (pack *Packer) Get(id int) (rect image.Rectangle) {
	var has bool
	if rect, has = pack.rects[id]; !has {
		rect = pack.rects[pack.nfId]
	}
	return
}

func (pack *Packer) Image() image.Image {
	return pack.pic
}
