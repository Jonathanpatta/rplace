package placeclone

import (
	"errors"
	"fmt"
	"time"
)

type Image struct {
	Pixels []*Pixel `json:"pixels,omitempty"`
	Rows   int      `json:"rows,omitempty"`
	Cols   int      `json:"height,omitempty"`
	Name   string   `json:"name,omitempty"`
}

type Pixel struct {
	Pk           string
	Sk           string
	Row          int    `json:"row"`
	Col          int    `json:"col"`
	Color        string `json:"color,omitempty"`
	Author       string `json:"author,omitempty"`
	LastModified int64  `json:"last_modified,omitempty"`
}

func GetSortKey(row int, col int) string {
	return fmt.Sprintf("%v#%v", row, col)
}

func (i *Image) UpdatePixel(row int, col int, color string, author string) (*Pixel, error) {
	pixel := &Pixel{
		Pk:           i.Name,
		Sk:           GetSortKey(row, col),
		Row:          row,
		Col:          col,
		Color:        color,
		Author:       author,
		LastModified: time.Now().Unix(),
	}

	ok, err := i.IsValidPixel(pixel)

	if !ok {
		return nil, err
	}

	i.Pixels[(row*i.Rows)+col] = pixel
	return pixel, nil
}

func (i *Image) UpdatePixelFromObject(p *Pixel) (*Pixel, error) {
	return i.UpdatePixel(p.Row, p.Col, p.Color, p.Author)
}

func (i *Image) IsValidPixel(p *Pixel) (bool, error) {
	if !i.WithinBounds(p) {
		return false, errors.New("pixel out of bounds")
	}

	return true, nil
}

func (i *Image) WithinBounds(p *Pixel) bool {
	if p.Row < i.Rows && p.Col < i.Cols && p.Row >= 0 && p.Col >= 0 {
		return true
	}

	return false
}

func NewImage(name string, width int, height int) *Image {
	return &Image{
		Pixels: make([]*Pixel, height*width),
		Rows:   width,
		Cols:   height,
		Name:   name,
	}
}
