package dosfont

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/lunixbochs/struc"
)

type fileFormat int

const (
	other fileFormat = iota
	pe
	ne
)

// Resource resource table entry
type Resource struct {
	Start uint16
	Size  uint16
	Type  uint16
}

// MZ dos resource container
type MZ struct {
	Signature          uint16        `struc:"uint16,little"`
	BytesInLastBlock   uint16        `struc:"uint16,little"`
	BlocksInFile       uint16        `struc:"uint16,little"`
	NumRelocs          uint16        `struc:"uint16,little"`
	HeaderParagraphs   uint16        `struc:"uint16,little"`
	MinExtraParagraphs uint16        `struc:"uint16,little"`
	MaxExtraParagraphs uint16        `struc:"uint16,little"`
	Ss                 uint16        `struc:"uint16,little"`
	Sp                 uint16        `struc:"uint16,little"`
	Checksum           uint16        `struc:"uint16,little"`
	IP                 uint16        `struc:"uint16,little"`
	Cs                 uint16        `struc:"uint16,little"`
	RelocTableOffset   uint16        `struc:"uint16,little"`
	OverlayNumber      uint16        `struc:"uint16,little"`
	Unknown            []byte        `struc:"[32]pad"`
	COFFHeaderOffset   uint32        `struc:"uint32,little"` // at 0x3c
	format             fileFormat    `struc:"skip"`
	resources          []Resource    `struc:"skip"`
	r                  io.ReadSeeker `struc:"skip"`
}

const headerSize int64 = 64

const fileIdentifier = 0x5a4D

// ErrHeaderToShort error in case the mz header is too short
var ErrHeaderToShort = errors.New("Error MZ header too short")

// ErrNoMZFile the passed reader doesn't contain the correct header
var ErrNoMZFile = errors.New("Error no MZ file")

// ErrInvalidOffset the resource table contains invalid offsets
var ErrInvalidOffset = errors.New("Error MZ file contains invalid offset")

// ReadMZ reads the MZ file from the passed reader
func ReadMZ(r io.ReadSeeker) (*MZ, error) {
	var mz MZ
	mz.r = r

	err := mz.readHeader()
	if err != nil {
		return nil, err
	}

	err = mz.identifyFileFormat()
	if err != nil {
		return nil, err
	}

	if mz.format != ne {
		return nil, fmt.Errorf("This file format %d is currently not supported", mz.format)
	}

	err = mz.readNEResourceTable()
	if err != nil {
		return nil, err
	}

	return &mz, nil
}

func (mz *MZ) readHeader() error {
	var buf bytes.Buffer

	n, err := io.CopyN(&buf, mz.r, headerSize)
	if err != nil {
		return err
	}
	if n != headerSize {
		return ErrHeaderToShort
	}

	err = struc.Unpack(&buf, mz)
	if err != nil {
		return err
	}

	if mz.Signature != fileIdentifier {
		return ErrNoMZFile
	}

	return nil
}

func (mz *MZ) identifyFileFormat() error {
	// goto COFF header
	_, err := mz.r.Seek(int64(mz.COFFHeaderOffset), io.SeekStart)
	if err != nil {
		return err
	}

	// read preamble
	var data [4]byte
	_, err = mz.r.Read(data[:])
	if err != nil {
		return err
	}

	// find file format
	if "NE" == string(data[0:2]) {
		mz.format = ne
	} else if "PE\x00\x00" == string(data[:]) {
		mz.format = pe
	}

	return nil
}

func (mz *MZ) readNEResourceTable() error {
	// goto COFF header
	_, err := mz.r.Seek(int64(mz.COFFHeaderOffset+0x24), io.SeekStart)
	if err != nil {
		return err
	}

	// find rtable
	var rtableStart uint16
	err = binary.Read(mz.r, binary.LittleEndian, &rtableStart)
	if err != nil {
		return err
	}
	rtable := int64(mz.COFFHeaderOffset + uint32(rtableStart))

	// go to table start
	_, err = mz.r.Seek(rtable, io.SeekStart)
	if err != nil {
		return err
	}

	// read shift value
	var rtableShift uint16
	err = binary.Read(mz.r, binary.LittleEndian, &rtableShift)
	if err != nil {
		return err
	}

	for {
		// Read Type
		var rtype uint16
		err = binary.Read(mz.r, binary.LittleEndian, &rtype)
		if err != nil {
			return err
		}

		// check for end of the resource table
		if rtype == 0 {
			break
		}

		// Read count
		var count uint16
		err = binary.Read(mz.r, binary.LittleEndian, &count)
		if err != nil {
			return err
		}

		// SKIP 4 bytes reserved
		_, err = mz.r.Seek(4, io.SeekCurrent)
		if err != nil {
			return err
		}

		for i := 0; i < int(count); i++ {
			// read start and size
			var start uint16
			err = binary.Read(mz.r, binary.LittleEndian, &start)
			if err != nil {
				return err
			}
			var size uint16
			err = binary.Read(mz.r, binary.LittleEndian, &size)
			if err != nil {
				return err
			}

			// shift values
			start = start << rtableShift
			size = size << rtableShift

			if start < 0 || size < 0 {
				return ErrInvalidOffset
			}

			// add fonts
			mz.resources = append(mz.resources, Resource{
				Start: start,
				Size:  size,
				Type:  rtype,
			})

			// flags, name/id, 4 bytes reserved
			_, err = mz.r.Seek(8, io.SeekCurrent)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Resources filters for resources of passed type, that are contained inside the MZ file
func (mz *MZ) Resources(filterType uint16) []Resource {
	var filtered []Resource

	for _, resource := range mz.resources {
		if resource.Type == filterType {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}
