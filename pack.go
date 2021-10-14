package rectpack

import (
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"sort"
)

// This texture packer algorithm is based on this project
// https://github.com/TeamHypersomnia/rectpack2D

var (
	ErrNoEmptySpace       = errors.New("Couldn't find an empty space")
	ErrSplitFailed        = errors.New("Split failed")
	ErrGrowthFailed       = errors.New("A previously added texture failed to be added after packer growth")
	ErrUnsupportedSaveExt = errors.New("Unsupported save filename extension")
	ErrNotPacked          = errors.New("Packer must be packed")
	ErrNotFoundNoDefault  = errors.New("Id doesn't exist and a default sprite wasn't specified")
	ErrAlreadyPacked      = errors.New("Pack has already been called for this packer")
)

type PackFlags uint8
type CreateFlags uint8

type PackerCfg struct {
	Flags CreateFlags
}

type Packer struct {
	cfg         PackerCfg
	bounds      image.Rectangle
	emptySpaces []image.Rectangle
	queued      []queuedData
	rects       map[int]image.Rectangle
	images      map[int]*image.RGBA
	pic         *image.RGBA
	nfId        int
	packed      bool
}

// Creates a new packer instance
func NewPacker(cfg PackerCfg) (pack *Packer) {
	bounds := rect(0, 0, 0, 0)
	pack = &Packer{
		cfg:         cfg,
		bounds:      bounds,
		emptySpaces: []image.Rectangle{},
		rects:       make(map[int]image.Rectangle),
		images:      make(map[int]*image.RGBA),
		queued:      make([]queuedData, 0),
		nfId:        -1,
	}
	return
}

// Inserts PictureData into the packer
func (pack *Packer) Insert(id int, pic *image.RGBA) {
	pack.queued = append(pack.queued, queuedData{id: id, pic: pic})
}

// Automatically parse and insert image from file.
func (pack *Packer) InsertFromFile(id int, filename string) (err error) {
	var (
		file *os.File
		img  image.Image
		rgba *image.RGBA
	)

	if file, err = os.Open(filename); err != nil {
		return err
	}
	defer file.Close()

	if img, _, err = image.Decode(file); err != nil {
		return err
	}

	switch i := img.(type) {
	case *image.RGBA:
		rgba = i
	default:
		r := i.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(rgba, rgba.Bounds(), i, r.Min, draw.Src)
	}

	pack.Insert(id, rgba)

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
func (pack *Packer) Pack() (err error) {
	if pack.packed {
		return ErrAlreadyPacked
	}

	// sort queued images largest to smallest
	sort.Slice(pack.queued, func(i, j int) bool {
		return area(pack.queued[i].pic.Bounds()) > area(pack.queued[j].pic.Bounds())
	})

	for i, data := range pack.queued {
		var (
			bounds   = data.pic.Bounds()
			_, found = pack.find(bounds)
		)

		if !found {
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
				var (
					rect = pack.rects[id]
				)
				pack.pic.Set(x+rect.Min.X, y+rect.Min.Y, pic.At(x, y))
			}
		}
	}
	pack.queued = nil
	pack.emptySpaces = nil
	pack.images = nil
	pack.packed = true

	return
}

// Saves the internal texture as a file on disk, the output type is defined by the filename extension
func (pack *Packer) Save(filename string) (err error) {
	if !pack.packed {
		return ErrNotPacked
	}

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

// Sets the default Id for the packer
//		If an id doesn't exist in the packer when 'Get' is called, the packer will return this sprite instead.
func (pack *Packer) SetDefaultId(id int) {
	pack.nfId = id
}

// Returns the subimage bounds from the given id
func (pack *Packer) Get(id int) (rect image.Rectangle) {
	if !pack.packed {
		panic(ErrNotPacked)
	}

	var has bool
	if rect, has = pack.rects[id]; !has {
		if pack.nfId == -1 {
			panic(ErrNotFoundNoDefault)
		}
		rect = pack.rects[pack.nfId]
	}
	return
}

// Returns the subimage, as a copy, from the given id
func (pack *Packer) SubImage(id int) (img *image.RGBA) {
	if !pack.packed {
		panic(ErrNotPacked)
	}

	r := pack.Get(id)
	i := pack.pic.PixOffset(r.Min.X, r.Min.Y)
	return &image.RGBA{
		Pix:    pack.pic.Pix[i:],
		Stride: pack.pic.Stride,
		Rect:   image.Rect(0, 0, r.Dx(), r.Dy()),
	}
}

// Returns the entire packed image
func (pack *Packer) Image() *image.RGBA {
	if !pack.packed {
		panic(ErrNotPacked)
	}
	return pack.pic
}
