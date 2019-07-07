package main

import (
	"bytes"
	"encoding/binary"
)

type dxt3Block struct {
	Alpha   uint64
	Color1  uint16
	Color2  uint16
	Indices uint32
}

func processDXT5(data *[]uint8, width uint16, height uint16) (*[]bgra, error) {
	numPixels := uint32(width) * uint32(height)

	//for j, i := range *data {
	//	fmt.Printf("%x ", i)
	//	if (j+1)%4 == 0 {
	//		fmt.Printf("\n")
	//	}
	//}

	//log.Printf("processDXT5: %v * %v = %v\n", width, height, numPixels)

	blocks := make([]dxt3Block, len(*data)/16)

	reader := bytes.NewBuffer(*data)
	if err := binary.Read(reader, binary.LittleEndian, &blocks); err != nil {
		return nil, err
	}

	//for _, x := range blocks {
	//	fmt.Printf("%08x %08x %016x\n", (x.Alpha >> 32) & 0xFFFFFFFF, x.Alpha & 0xFFFFFFFF, x.Alpha)
	//}

	pixels := make([]bgra, numPixels)

	numHorizBlocks := width >> 2
	numVertBlocks := height >> 2

	//log.Printf("processDXT5: %v %v", numHorizBlocks, numVertBlocks)

	var y uint16
	var x uint16

	for y = 0; y < numVertBlocks; y++ {
		for x = 0; x < numHorizBlocks; x++ {
			block := blocks[y*numHorizBlocks+x]

			processDXT5Block(&pixels, &block, x*4, y*4, width)
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

func processDXT5Block(pixelsPtr *[]bgra, dxt3Block *dxt3Block, blockX uint16, blockY uint16, width uint16) {
	pixels := *pixelsPtr
	indices := dxt3Block.Indices
	blockAlpha := dxt3Block.Alpha

	var colors [4]bgra
	var alphas [8]uint8

	block := dxtColor{
		color1: dxt3Block.Color1,
		color2: dxt3Block.Color2,
	}

	processDXTColor(&colors, &block, false, false)
	//fmt.Printf("%04x %04x | ", block.color1, block.color2)
	//for _, i := range colors {
	//	fmt.Printf("%02x%02x%02x ", i.r, i.g, i.b)
	//}
	//fmt.Printf("\n")

	//fmt.Printf("%016x ", blockAlpha)

	alphas[0] = uint8((blockAlpha >> 0) & 0xFF)
	alphas[1] = uint8((blockAlpha >> 8) & 0xFF)
	blockAlpha >>= 16

	var i uint
	if alphas[0] > alphas[1] {
		for i = 2; i < 8; i++ {
			first := (8 - i) * uint(alphas[0])
			second := (i - 1) * uint(alphas[1])
			total := (first + second) / 7
			// fmt.Printf("%04x %04x %04x ", first, second, total)
			alphas[i] = uint8(total)
		}
	} else {
		for i = 2; i < 6; i++ {
			first := (6 - i) * uint(alphas[0])
			second := (i - 1) * uint(alphas[1])
			total := (first + second) / 5
			// fmt.Printf("%04x %04x %04x ", first, second, total)
			alphas[i] = uint8(total)
		}
		alphas[6] = 0x00
		alphas[7] = 0xFF
		// fmt.Printf("---- ---- 0000 ---- ---- 00ff ")
	}

	//for i = 0; i < 8; i++ {
	//	fmt.Printf("%02x ", alphas[i])
	//}
	//fmt.Printf("%016x\n", blockAlpha)

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

			alphaIndex := blockAlpha & 7
			pixel.a = alphas[alphaIndex]

			//fmt.Printf("%02x_%02x ", alphaIndex, pixel.a)

			pixels[curPixel] = pixel

			curPixel++
			indices >>= 2
			blockAlpha >>= 3
		}
	}

	//fmt.Printf("\n")
}
