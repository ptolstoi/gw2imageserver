package gw2dat

import (
	"encoding/binary"
	"io"
	"log"
	"os"
)

type GW2DatReader interface {
	Close()
	Header() GW2DatHeader
	MFTHeader() MFTHeader
}

type gw2DatReader struct {
	file      *os.File
	header    GW2DatHeader
	mftHeader MFTHeader
}

func NewGW2DatReader(filePath string) (GW2DatReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	reader := gw2DatReader{
		file: file,
	}

	if err := binary.Read(file, binary.LittleEndian, &reader.header); err != nil {
		return nil, err
	}

	if _, err := file.Seek(int64(reader.header.MFTOffset), io.SeekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &reader.mftHeader); err != nil {
		return nil, err
	}

	if _, err := file.Seek(int64(reader.header.MFTOffset), io.SeekStart); err != nil {
		return nil, err
	}
	mftEntries := make([]MFTEntry, reader.mftHeader.NumberOfEntries)
	if err := binary.Read(file, binary.LittleEndian, &mftEntries); err != nil {
		return nil, err
	}

	i := 0
	c := 0
	for c < 10 {
		i++
		if mftEntries[i].Size == 0 {
			continue
		}

		log.Printf("%v: %#v", i, mftEntries[i])

		if i == 0 {
			continue
		}

		c++

		if _, err := file.Seek(int64(mftEntries[i].Offset), io.SeekStart); err != nil {
			return nil, err
		}
		arr := make([]byte, 10)
		if err := binary.Read(file, binary.LittleEndian, &arr); err != nil {
			return nil, err
		}

		log.Printf("%#v %v", arr, string(arr))
	}

	return &reader, nil
}

func (reader *gw2DatReader) Close() {
	_ = reader.file.Close()
}

func (reader *gw2DatReader) Header() GW2DatHeader {
	return reader.header
}

func (reader *gw2DatReader) MFTHeader() MFTHeader {
	return reader.mftHeader
}
