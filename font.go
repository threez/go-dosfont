package dosfont

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"strings"

	"golang.org/x/image/font/basicfont"

	"github.com/lunixbochs/struc"
)

// location of the font header
const fontHeader = 0x76

// type of font resource in mz file
const fontType = 0x8008

// Font contains all the PC Screen Font (PSF) meta data and font face.
type Font struct {
	Version         uint16 `struc:"uint16,little"`
	Size            uint32 `struc:"uint32,little"`
	Copyright       string `struc:"[60]byte"`
	Type            uint16 `struc:"uint16,little"`
	Points          uint16 `struc:"uint16,little"`
	VertRes         uint16 `struc:"uint16,little"`
	HorizRes        uint16 `struc:"uint16,little"`
	Ascent          uint16 `struc:"uint16,little"`
	InternalLeading uint16 `struc:"uint16,little"`
	ExternalLeading uint16 `struc:"uint16,little"`
	Italic          bool
	Underline       bool
	StrikeOut       bool
	Weight          uint16 `struc:"uint16,little"`
	CharSet         uint8
	PixWidth        uint16 `struc:"uint16,little"`
	PixHeight       uint16 `struc:"uint16,little"`
	PitchAndFamily  uint8
	AvgWidth        uint16 `struc:"uint16,little"`
	MaxWidth        uint16 `struc:"uint16,little"`
	FirstChar       uint8
	LastChar        uint8
	DefaultChar     uint8
	BreakChar       uint8
	WidthBytes      uint16 `struc:"uint16,little"`
	Device          uint16 `struc:"uint16,little"`
	FaceData        uint16 `struc:"uint16,little"`
	BitsPointer     uint16 `struc:"uint16,little"`
	BitsOffset      uint16 `struc:"uint16,little"`
	Reserved        uint8
	Name            string `struc:"skip"`
	basicfont.Face  `struc:"skip"`
}

// ErrFontHeaderTooShort error in case the header is to short
var ErrFontHeaderTooShort = errors.New("Error psf header too short")

// OpenFonts opens the file under given location and returns contained fonts
func OpenFonts(path string) ([]Font, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open font file %s: %v", path, err)
	}
	defer file.Close()

	return ReadFonts(file)
}

// ReadFonts reads font faces from a read seeker
func ReadFonts(r io.ReadSeeker) ([]Font, error) {
	mz, err := ReadMZ(r)
	if err != nil {
		return nil, err
	}

	fontResources := mz.Resources(fontType)
	fonts := make([]Font, len(fontResources))

	for i, font := range fontResources {
		// seek to font
		_, err := r.Seek(int64(font.Start), io.SeekStart)
		if err != nil {
			return nil, err
		}

		// read and parse header
		var buf bytes.Buffer
		n, err := io.CopyN(&buf, r, fontHeader)
		if err != nil {
			return nil, err
		}
		if n != fontHeader {
			return nil, ErrFontHeaderTooShort
		}

		err = struc.Unpack(&buf, &fonts[i])
		if err != nil {
			return nil, err
		}

		// seek to font Name and fix copyright
		fonts[i].Copyright = strings.Trim(fonts[i].Copyright, " ")
		if fonts[i].BitsPointer > 0 {
			_, err = r.Seek(int64(font.Start+fonts[i].BitsPointer), io.SeekStart)
			if err != nil {
				return nil, err
			}
			fonts[i].Name, err = bufio.NewReader(r).ReadString(0x00)
			if err != nil {
				return nil, err
			}
			// remove delimiter (null byte)
			fonts[i].Name = fonts[i].Name[:len(fonts[i].Name)-1]
		}

		// read chars
		var char Resource
		if fonts[i].Version == 0x200 {
			char.Start = 0x76
			char.Size = 4
		} else {
			char.Start = 0x94
			char.Size = 6
		}

		glyphCount := int(fonts[i].LastChar - fonts[i].FirstChar)

		// Advance is the glyph advance, in pixels.
		fonts[i].Face.Advance = int(fonts[i].AvgWidth)
		// Width is the glyph width, in pixels.
		fonts[i].Face.Width = int(fonts[i].PixWidth)
		// Height is the inter-line height, in pixels.
		fonts[i].Face.Height = int(fonts[i].PixHeight)
		// Ascent is the glyph ascent, in pixels.
		fonts[i].Face.Ascent = int(fonts[i].PixHeight)
		// Descent is the glyph descent, in pixels.
		fonts[i].Face.Descent = 0
		// Left is the left side bearing, in pixels. A positive value means that
		// all of a glyph is to the right of the dot.
		fonts[i].Face.Left = 0

		// Mask contains all of the glyph masks. Its width is typically the Face's
		// Width, and its height a multiple of the Face's Height.
		fontmask := &image.Alpha{
			Stride: fonts[i].Face.Width,
			Rect: image.Rectangle{
				Max: image.Point{
					fonts[i].Face.Width, glyphCount * fonts[i].Face.Height,
				},
			},
			Pix: make([]byte, glyphCount*fonts[i].Face.Width*fonts[i].Face.Height),
		}
		fonts[i].Face.Mask = fontmask

		// Mask image.Image
		// Ranges map runes to sub-images of Mask. The rune ranges must not
		// overlap, and must be in increasing rune order.
		fonts[i].Face.Ranges = []basicfont.Range{
			{
				Low:    rune(fonts[i].FirstChar),
				High:   rune(fonts[i].LastChar),
				Offset: 0,
			},
		}

		for j := 0; j < glyphCount; j++ {
			// address of the glyph in the mask
			maskOffsetAddress := j * (fonts[i].Face.Height) * fonts[i].Face.Width

			charPos := int(font.Start) + int(char.Start) + int(char.Size)*j
			_, err = r.Seek(int64(charPos), io.SeekStart)
			if err != nil {
				return nil, err
			}

			// read width
			var width uint16
			err = binary.Read(r, binary.LittleEndian, &width)
			if err != nil {
				return nil, err
			}

			// read offset
			var offset int
			if char.Size == 4 {
				var off uint16
				err = binary.Read(r, binary.LittleEndian, &off)
				offset = int(off)
			} else {
				var off uint32
				err = binary.Read(r, binary.LittleEndian, &off)
				offset = int(off)
			}
			if err != nil {
				return nil, err
			}

			// read data
			charBitPos := int(font.Start) + offset
			_, err = r.Seek(int64(charBitPos), io.SeekStart)
			if err != nil {
				return nil, err
			}
			mask := uint8(0x80)
			pk := int((width + 7) / 8)
			// read column
			for column := 0; column < pk; column++ {
				// read row
				for row := 0; row < int(fonts[i].Face.Height); row++ {
					// read byte
					var b uint8
					err = binary.Read(r, binary.LittleEndian, &b)
					if err != nil {
						return nil, err
					}
					rowAddr := maskOffsetAddress + row*(fonts[i].Face.Width)

					for bits := 0; bits < 8; bits++ {
						// don't try to put filler bits into mask
						if bits+column*8 >= fonts[i].Face.Width {
							break
						}
						addr := rowAddr + bits + column*8

						// check if first bit is set
						if b&mask == mask {
							fontmask.Pix[addr] = 0xff
							// fmt.Print("1")
						} else {
							fontmask.Pix[addr] = 0x00
							// fmt.Print("0")
						}
						b <<= 1 // goto next bit
					}
					// fmt.Print(" (", column, row, rowAddr, ")\n")
				}
			}
			// fmt.Print("--- \n\n")
		}
	}

	return fonts, nil
}
