package dosfont

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestFontRendering(t *testing.T) {
	for _, path := range []string{"fonts/DOSLike.FON", "fonts/NEW1252.FON"} {
		fonts, err := OpenFonts(path)
		if err != nil {
			t.Fatal("Failed to read fonts:", err)
		}

		// create test rendering of fonts
		img := image.NewRGBA(image.Rect(0, 0, 1000, 300))
		for i := 0; i < len(fonts); i++ {
			t.Logf("Font: %s (%s) [%dx%d]", fonts[i].Name, fonts[i].Copyright, fonts[i].PixWidth, fonts[i].PixHeight)
			off := i*30 + 30

			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(color.RGBA{0x00, 0x00, 0x00, 255}),
				Face: &fonts[i].Face,
				Dot:  fixed.Point26_6{fixed.Int26_6(30 * 64), fixed.Int26_6(off * 64)},
			}
			d.DrawString("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!§$%&/()=?äöüÖÄÜß,.-_:;#+'*`?^°<>")
		}

		// render png of image
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}

		// compare
		reference, err := ioutil.ReadFile(path + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Compare(buf.Bytes(), reference) != 0 {
			t.Errorf("Rendering with font %s failed", path)
		}
	}
}
