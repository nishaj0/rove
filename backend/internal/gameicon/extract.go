// Package gameicon reads a safe, display-sized PNG from an executable icon.
package gameicon

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/tc-hib/winres"
)

const maxSide = 512

// ExtractDataURL returns the largest PNG image embedded in the first icon
// group. Many modern Windows executables include a PNG 256x256 icon; older
// bitmap-only icons simply fall back to the generated cover instead of making
// game registration fail.
func ExtractDataURL(executablePath string) (string, error) {
	f, err := os.Open(executablePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	rs, err := winres.LoadFromEXESingleType(f, winres.RT_GROUP_ICON)
	if err != nil {
		return "", err
	}
	var best image.Image
	rs.WalkType(winres.RT_GROUP_ICON, func(id winres.Identifier, _ uint16, _ []byte) bool {
		icon, iconErr := rs.GetIcon(id)
		if iconErr != nil {
			return true
		}
		var ico bytes.Buffer
		if icon.SaveICO(&ico) != nil {
			return true
		}
		for _, candidate := range pngImagesFromICO(ico.Bytes()) {
			if best == nil || candidate.Bounds().Dx()*candidate.Bounds().Dy() > best.Bounds().Dx()*best.Bounds().Dy() {
				best = candidate
			}
		}
		return best == nil
	})
	if best == nil {
		return "", errors.New("executable has no PNG icon resource")
	}
	if best.Bounds().Dx() > maxSide || best.Bounds().Dy() > maxSide {
		best = scaleDown(best, maxSide)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, best); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

func pngImagesFromICO(data []byte) []image.Image {
	if len(data) < 6 {
		return nil
	}
	count := int(data[4]) | int(data[5])<<8
	images := []image.Image{}
	for i := 0; i < count; i++ {
		off := 6 + 16*i
		if off+16 > len(data) {
			break
		}
		size := int(data[off+8]) | int(data[off+9])<<8 | int(data[off+10])<<16 | int(data[off+11])<<24
		start := int(data[off+12]) | int(data[off+13])<<8 | int(data[off+14])<<16 | int(data[off+15])<<24
		if size <= 0 || start < 0 || start+size > len(data) {
			continue
		}
		payload := data[start : start+size]
		if len(payload) < 8 || !bytes.Equal(payload[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			continue
		}
		img, err := png.Decode(bytes.NewReader(payload))
		if err == nil {
			images = append(images, img)
		}
	}
	return images
}

func scaleDown(src image.Image, limit int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= limit && h <= limit {
		return src
	}
	if w >= h {
		h = h * limit / w
		w = limit
	} else {
		w = w * limit / h
		h = limit
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, src.At(b.Min.X+x*b.Dx()/w, b.Min.Y+y*b.Dy()/h))
		}
	}
	return dst
}

func SourceForDataURL(data string) string {
	if data == "" {
		return "generated"
	}
	return "manual"
}
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
