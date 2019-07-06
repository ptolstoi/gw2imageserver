package main

import (
	"bytes"
	"encoding/binary"
)

type dxtColor struct {
	color1 uint16
	color2 uint16
}

type dxt1Block struct {
	Color1  uint16
	Color2  uint16
	Indices uint32
}

func processDXT1(data *[]uint8, width uint16, height uint16) (*[]bgra, error) {
	numPixels := uint32(width) * uint32(height)

	// log.Printf("processDXT1: %v * %v = %v\n", width, height, numPixels)

	blocks := make([]dxt1Block, len(*data)/8)

	reader := bytes.NewBuffer(*data)
	if err := binary.Read(reader, binary.LittleEndian, &blocks); err != nil {
		return nil, err
	}

	pixels := make([]bgra, numPixels)

	numHorizBlocks := width >> 2
	numVertBlocks := height >> 2

	// log.Printf("processDXT1: %v %v", numHorizBlocks, numVertBlocks)

	var y uint16
	var x uint16

	for y = 0; y < numVertBlocks; y++ {
		for x = 0; x < numHorizBlocks; x++ {
			block := blocks[y*numHorizBlocks+x]

			processDXT1Block(&pixels, &block, x*4, y*4, width)
		}
	}

	// for y = 0; y < height; y++ {
	// 	for x = 0; x < width; x++ {
	// 		fmt.Printf("%+v ", pixels[y*width+x])
	// 	}
	// 	fmt.Printf("\n")
	// }

	return &pixels, nil
}

func processDXT1Block(pixelsPtr *[]bgra, dxt1Block *dxt1Block, blockX uint16, blockY uint16, width uint16) {
	pixels := *pixelsPtr
	indices := dxt1Block.Indices
	var colors [4]bgra

	block := dxtColor{
		color1: dxt1Block.Color1,
		color2: dxt1Block.Color2,
	}

	processDXTColor(&colors, &block, true, true)

	var y uint16
	var x uint16
	for y = 0; y < 4; y++ {
		curPixel := uint(blockY+y)*uint(width) + uint(blockX)

		for x = 0; x < 4; x++ {
			pixel := pixels[curPixel]
			index := indices & 3

			pixel.r = colors[index].r
			pixel.g = colors[index].g
			pixel.b = colors[index].b
			pixel.a = colors[index].a

			pixels[curPixel] = pixel

			curPixel++
			indices >>= 2
		}
	}
}
