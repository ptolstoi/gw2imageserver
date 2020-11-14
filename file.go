package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ptolstoi/gw2imageserver/textureInflater"
)

const (
	fccATEX = "\x41\x54\x45\x58"
	fccATTX = "\x41\x54\x54\x58"
	fccATEC = "\x41\x54\x45\x43"
	fccATEP = "\x41\x54\x45\x50"
	fccATEU = "\x41\x54\x45\x55"
	fccATET = "\x41\x54\x45\x54"

	//fccATEXn uint32 = 0x58455441
	//fccATTXn uint32 = 0x58545441
	//fccATECn uint32 = 0x43455441
	//fccATEPn uint32 = 0x50455441
	//fccATEUn uint32 = 0x55455441
	//fccATETn uint32 = 0x54455441
)

type file struct {
	file         string
	content      []byte
	fileType     string
	lastModified time.Time
}

func (app *app) fetchFile(fileID string) (*file, error) {
	// https://render.guildwars2.com/file/BFD2CB5A0604A4425DF9CD22DF0F40C4E0AE9AAA/602790.jpg
	// https://render.guildwars2.com/file/BFD2CB5A0604A4425DF9CD22DF0F40C4E0AE9AAA/602790.png
	// http://assetcdn.101.arenanetworks.com/program/101/1/0/602790
	// authCookie=access=/latest/*!/manifest/program/*!/program/*~md5=4e51ad868f87201ad93e428ff30c6691
	url := fmt.Sprintf("http://assetcdn.101.ArenaNetworks.com/program/101/1/0/%v", fileID)

	log.Printf("[fetchFile] fetching %v", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.AddCookie(&http.Cookie{
		Name:  "authCookie",
		Value: "access=/latest/*!/manifest/program/*!/program/*~md5=4e51ad868f87201ad93e428ff30c6691",
	})

	response, err := app.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	file := file{
		file:         fileID,
		fileType:     "uncompressed",
		lastModified: time.Now().UTC(),
		content:      body,
	}

	return &file, nil
}

func (app *app) noImageFileInCache(fileID string, fileType string) (*file, error) {
	uncompressedFile, err := app.getFileFromCache(fileID, "uncompressed")

	if uncompressedFile == nil && err == nil {
		uncompressedFile, err = app.fetchFile(fileID)

		if err == nil && uncompressedFile != nil {
			err = app.saveFileToCache(uncompressedFile)
		}
	}
	if err != nil {
		return nil, err
	}

	data := uncompressedFile.content

	log.Printf("[noFileInCache] file found: file=%v type=%v length=%v lastModified=%v", uncompressedFile.file, uncompressedFile.fileType, len(uncompressedFile.content), uncompressedFile.lastModified)
	// log.Printf("\n%s", hex.Dump(data[0:(16*10)]))

	if err := checkHeader(data); err != nil {
		return nil, err
	}

	format := string(data[4:8])
	width := binary.LittleEndian.Uint16(data[8:10])
	height := binary.LittleEndian.Uint16(data[10:12])
	numBlocks := uint32((width+3)>>2) * uint32((height+3)>>2)

	if format == textureInflater.FccDXT1 {
		numBlocks *= 8
	} else if format == textureInflater.FccDXT5 {
		numBlocks *= 16
	} else {
		return nil, fmt.Errorf("unknown ATEX texture format: %v", format)
	}

	//log.Printf("width: %v, height: %v, format: %v, numBlocks: %v", width, height, format, numBlocks)

	imgRaw, err := textureInflater.Inflate(data, width, height)

	if err != nil {
		return nil, err
	}

	if fileType == "png" {
		return app.saveImageAsPNG(fileID, &imgRaw)
	}

	return nil, fmt.Errorf("unknown file type")
}

func checkHeader(data []byte) error {
	if len(data) < 0x10 {
		return fmt.Errorf("data too small")
	}

	fourCC := string(data[0:4])

	if fourCC != fccATEX && fourCC != fccATTX && fourCC != fccATEP && fourCC != fccATEU && fourCC != fccATEC && fourCC != fccATET {
		return fmt.Errorf("unknown format: %v", fourCC)
	}

	compression := string(data[4:8])

	if compression != textureInflater.FccDXT1 && compression != textureInflater.FccDXT5 {
		return fmt.Errorf("unknown compression: %v", compression)
	}

	return nil
}

func (app *app) saveImageAsPNG(fileID string, imgRaw *image.Image) (*file, error) {
	buffer := new(bytes.Buffer)

	encoder := png.Encoder{
		CompressionLevel: png.BestCompression,
	}

	if err := encoder.Encode(buffer, *imgRaw); err != nil {
		return nil, err
	}

	newFile := file{
		content:      buffer.Bytes(),
		file:         fileID,
		lastModified: time.Now().UTC(),
		fileType:     "png",
	}

	if err := app.saveFileToCache(&newFile); err != nil {
		return nil, err
	}

	return &newFile, nil
}
