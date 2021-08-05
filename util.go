package rectpack

import (
	"image"
)

type queuedData struct {
	id  int
	pic *image.RGBA
}

// container for the leftover space after split
type createdSplits struct {
	hasSmall, hasBig bool
	count            int
	smaller, bigger  image.Rectangle
}

// adds the given leftover spaces to this container
func splits(rects ...image.Rectangle) (s *createdSplits) {
	s = &createdSplits{
		count:    len(rects),
		hasSmall: true,
		smaller:  rects[0],
	}

	if s.count == 2 {
		s.hasBig = true
		s.bigger = rects[1]
	}

	return
}

// helper function to create rectangles
func rect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

func area(r image.Rectangle) int {
	return r.Dx() * r.Dy()
}

// helper to split existing space
func split(img, space image.Rectangle) (s *createdSplits, err error) {
	w := space.Dx() - img.Dx()
	h := space.Dy() - img.Dy()

	if w < 0 || h < 0 {
		return nil, ErrorSplitFailed
	} else if w == 0 && h == 0 {
		// perfectly fit case
		return &createdSplits{}, nil
	} else if w > 0 && h == 0 {
		r := rect(space.Min.X+img.Dx(), space.Min.Y, w, img.Dy())
		return splits(r), nil
	} else if w == 0 && h > 0 {
		r := rect(space.Min.X, space.Min.Y+img.Dy(), img.Dx(), h)
		return splits(r), nil
	}

	var smaller, larger image.Rectangle
	if w > h {
		smaller = rect(space.Min.X, space.Min.Y+img.Dy(), img.Dx(), h)
		larger = rect(space.Min.X+img.Dx(), space.Min.Y, w, space.Dy())
	} else {
		smaller = rect(space.Min.X+img.Dx(), space.Min.Y, w, img.Dy())
		larger = rect(space.Min.X, space.Min.Y+img.Dy(), space.Dx(), h)
	}

	return splits(smaller, larger), nil
}
