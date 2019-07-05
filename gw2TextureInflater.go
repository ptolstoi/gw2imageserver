package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	imageColor "image/color"
	"log"
)

type inflaterState struct {
	input     []uint32
	inputSize uint32
	inputPos  uint32

	head   uint32
	buffer uint32
	bits   uint8

	isEmpty bool
}

type format struct {
	flags           uint16
	pixelSizeInBits uint16
}

type fullFormat struct {
	*format

	nbObPixelBlocks    uint32
	bytesPerPixelBlock uint32
	bytesPerComponent  uint32
	hasTwoComponents   bool

	width  uint16
	height uint16
}

type bgra struct {
	b uint8
	g uint8
	r uint8
	a uint8
}

type DXT1Block struct {
	Color1  uint16
	Color2  uint16
	Indices uint32
}

const (
	ffColor            uint16 = 0x10
	ffAlpha            uint16 = 0x20
	ffDeducedAlphaComp uint16 = 0x40
	ffPlainComp        uint16 = 0x80
	ffBiColorComp      uint16 = 0x200
)

const (
	cfDecodeWhiteColor             = 0x01
	cfDecodeConstantAlphaFrom4Bits = 0x02
	cfDecodeConstantAlphaFrom8Bits = 0x04
	cfDecodePlainColor             = 0x08
)

func inflate(inputRaw []byte, outputSize uint32) (image.Image, error) {
	input := make([]uint32, len(inputRaw)/4)
	binary.Read(bytes.NewBuffer(inputRaw[:]), binary.LittleEndian, &input)

	state := inflaterState{
		input:     input,
		inputSize: uint32(len(input)),
		inputPos:  0,

		head:   0,
		bits:   0,
		buffer: 0,

		isEmpty: false,
	}

	// skip header
	log.Printf("skipping header")
	if err := state.needBits(32); err != nil {
		return nil, err
	}
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	log.Printf("reading format")
	// format
	if err := state.needBits(32); err != nil {
		return nil, err
	}
	formatFourCC := state.readBits(32)
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	log.Printf("fourmatFourCC: % X", formatFourCC)

	format := deductFormat(string(formatFourCC&0xFF) + string(formatFourCC&0xFF00>>8) + string(formatFourCC&0xFF0000>>16) + string(formatFourCC&0xFF000000>>24))
	aFullFormat := fullFormat{
		format: &format,
	}

	log.Printf("reading size")
	state.needBits(32)
	aFullFormat.width = uint16(state.readBits(16))
	state.dropBits(16)
	aFullFormat.height = uint16(state.readBits(16))
	state.dropBits(16)

	aFullFormat.nbObPixelBlocks = uint32((aFullFormat.width+3)/4) * uint32((aFullFormat.height+3)/4)
	aFullFormat.bytesPerPixelBlock = uint32(aFullFormat.pixelSizeInBits) * 4 * 4 / 8
	aFullFormat.hasTwoComponents = ((aFullFormat.flags & (ffPlainComp | ffColor | ffAlpha)) == (ffPlainComp | ffColor | ffAlpha)) || (aFullFormat.flags&ffBiColorComp) != 0
	aFullFormat.bytesPerComponent = aFullFormat.bytesPerPixelBlock
	if aFullFormat.hasTwoComponents {
		aFullFormat.bytesPerComponent = aFullFormat.bytesPerPixelBlock / 2
	}

	log.Printf("FullFormat: {flags:% 016b(%[1]v) pixelSizeInBits:%v} %+v",
		aFullFormat.flags,
		aFullFormat.pixelSizeInBits,
		aFullFormat)

	anOutputSize := aFullFormat.bytesPerPixelBlock * aFullFormat.nbObPixelBlocks

	log.Printf("anOutputSize: %v", anOutputSize)

	result := state.inflateData(aFullFormat, anOutputSize)

	var colors *[]bgra
	if formatFourCC == fccDXT1n {
		log.Printf("fccDXT1")
		var err error
		colors, err = processDXT1(&result, aFullFormat.width, aFullFormat.height)

		if err != nil {
			return nil, err
		}

		log.Printf("colors: %v", len(*colors))
	} else if formatFourCC == fccDXT5n {
		log.Printf("fccDXT5")
		return nil, fmt.Errorf("fccDXT5 not implemented yet")
	} else {
		return nil, fmt.Errorf("unknown formatFourCC: %08X", formatFourCC)
	}

	img := image.NewRGBA(image.Rect(0, 0, int(aFullFormat.width), int(aFullFormat.height)))
	var y uint16
	var x uint16
	for y = 0; y < aFullFormat.height; y++ {
		for x = 0; x < aFullFormat.width; x++ {
			index := uint(y)*uint(aFullFormat.width) + uint(x)
			color := (*colors)[index]

			img.SetRGBA(int(x), int(y), imageColor.RGBA{
				R: color.r,
				G: color.g,
				B: color.b,
				A: color.a,
			})
		}
	}

	return img, nil
}

func (state *inflaterState) inflateData(fullFormat fullFormat, outputSize uint32) []uint8 {
	ioOutputTab := make([]uint8, outputSize)

	state.head = 0
	state.bits = 0
	state.buffer = 0

	state.needBits(32)
	aDataSize := state.readBits(32)
	state.dropBits(32)

	state.needBits(32)
	aCompressionFlags := state.readBits(32)
	state.dropBits(32)

	aColorBitmap := make([]bool, fullFormat.nbObPixelBlocks)
	// aAlphaBitmap := make([]bool, fullFormat.nbObPixelBlocks)

	log.Printf("aDataSize: %v, aCompressionFlags: %032b", aDataSize, aCompressionFlags)

	if aCompressionFlags&cfDecodeWhiteColor != 0 {
		log.Printf("cfDecodeWhiteColor")
	}
	if aCompressionFlags&cfDecodeConstantAlphaFrom4Bits != 0 {
		log.Printf("cfDecodeConstantAlphaFrom4Bits")
	}
	if aCompressionFlags&cfDecodeConstantAlphaFrom8Bits != 0 {
		log.Printf("cfDecodeConstantAlphaFrom8Bits")
		// state.decodeConstantAlphaFrom8Bits(&aAlphaBitmap, fullFormat, &ioOutputTab)
	}
	if aCompressionFlags&cfDecodePlainColor != 0 {
		log.Printf("cfDecodePlainColor")
	}

	var i uint32
	if state.bits >= 32 {
		state.inputPos--
	}

	if ((fullFormat.flags&ffAlpha) != 0 && (fullFormat.flags&ffDeducedAlphaComp) == 0) || (fullFormat.flags&ffBiColorComp) != 0 {
		log.Printf("LOOP1")

		// if (((iFullFormat.format.flags & FF_ALPHA) && !(iFullFormat.format.flags & FF_DEDUCEDALPHACOMP)) || iFullFormat.format.flags & FF_BICOLORCOMP) {

		// 	std::cout << "LOOP1" << std::endl;

		// 	for (aLoopIndex = 0; aLoopIndex < aAlphaBitmap.size() && iState.inputPos < iState.inputSize; ++aLoopIndex) {
		// 		if (!aAlphaBitmap[aLoopIndex]) {
		// 			(*reinterpret_cast<uint32_t*>(&(ioOutputTab[iFullFormat.bytesPerPixelBlock * aLoopIndex]))) = iState.input[iState.inputPos];
		// 			++iState.inputPos;
		// 			if (iFullFormat.bytesPerComponent > 4) {
		// 				(*reinterpret_cast<uint32_t*>(&(ioOutputTab[iFullFormat.bytesPerPixelBlock * aLoopIndex + 4]))) = iState.input[iState.inputPos];
		// 				++iState.inputPos;
		// 			}
		// 		}
		// 	}
		// }
	}

	if (fullFormat.flags&ffColor) != 0 || (fullFormat.flags&ffBiColorComp) != 0 {
		aColorSize := uint32(len(aColorBitmap))
		log.Printf("LOOP2 %v %v", aColorSize, state.inputSize)
		for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
			if !aColorBitmap[i] {
				offset := fullFormat.bytesPerPixelBlock * i
				if fullFormat.hasTwoComponents {
					offset += fullFormat.bytesPerComponent
				}

				data := state.input[state.inputPos]

				// fmt.Printf("%08X\n", data)

				ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
				ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
				ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
				ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

				state.inputPos++
			}
		}

		if fullFormat.bytesPerComponent > 4 {
			// log.Printf("LOOP3")

			// for (aLoopIndex = 0; aLoopIndex < aColorBitmap.size() && iState.inputPos < iState.inputSize; ++aLoopIndex) {
			// 	if (!aColorBitmap[aLoopIndex]) {
			// 		uint32_t aOffset = iFullFormat.bytesPerPixelBlock * aLoopIndex + 4 + (iFullFormat.hasTwoComponents ? iFullFormat.bytesPerComponent : 0);
			// 		(*reinterpret_cast<uint32_t*>(&(ioOutputTab[aOffset]))) = iState.input[iState.inputPos];
			// 		++iState.inputPos;
			// 	}
			// }
			for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
				if !aColorBitmap[i] {
					offset := fullFormat.bytesPerPixelBlock*i + 4

					if fullFormat.hasTwoComponents {
						offset += fullFormat.bytesPerComponent
					}

					data := state.input[state.inputPos]
					// fmt.Printf("%08X\n", data)

					ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
					ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
					ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
					ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

					state.inputPos++
				}
			}
		}
	}

	// size := uint32(len(ioOutputTab))
	// for i = 0; i < size; i++ {
	// 	fmt.Printf("%02X", ioOutputTab[i])
	// 	if (i+1)%uint32(fullFormat.width) == 0 {
	// 		fmt.Printf("\n")
	// 	}
	// }
	// fmt.Printf("\n")

	return ioOutputTab
}

func (state *inflaterState) decodeConstantAlphaFrom8Bits(aAlphaBitmap *[]bool, fullFormat fullFormat, result *[]uint8) {
	// state.needBits(8)
	// aAlphaValueByte := uint64(state.readBits(8))
	// state.dropBits(8)

	// var aPixelBlockPos uint32 = 0

	// var aAlphaValue uint64 = aAlphaValueByte | uint64(aAlphaValueByte<<8)
	// var zero uint64

	// for aPixelBlockPos < fullFormat.nbObPixelBlocks {

	// }
}

func processDXT1(data *[]uint8, width uint16, height uint16) (*[]bgra, error) {

	numPixels := width * height

	blocks := make([]DXT1Block, len(*data)/8)

	reader := bytes.NewBuffer(*data)
	if err := binary.Read(reader, binary.LittleEndian, &blocks); err != nil {
		return nil, err
	}

	pixels := make([]bgra, numPixels)

	numHorizBlocks := width >> 2
	numVertBlocks := width >> 2

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

func processDXT1Block(pixelsPtr *[]bgra, block *DXT1Block, blockX uint16, blockY uint16, width uint16) {
	pixels := *pixelsPtr
	indices := block.Indices
	var colors [4]bgra

	processDXTColor(&colors, block, true)

	var y uint16
	var x uint16
	for y = 0; y < 4; y++ {
		curPixel := uint(blockY+y)*uint(width) + uint(blockX) // TODO test if correct

		for x = 0; x < 4; x++ {
			pixel := pixels[curPixel]
			index := indices & 3

			pixel.r = colors[index].r
			pixel.g = colors[index].b
			pixel.b = colors[index].b
			pixel.a = colors[index].a

			pixels[curPixel] = pixel

			curPixel++
			indices >>= 2
		}
	}
}

func processDXTColor(pixel *[4]bgra, block *DXT1Block, isDXT1 bool) {
	red1 := (block.Color1 & 0xF800) >> 11
	green1 := (block.Color1 & 0x07E0) >> 5
	blue1 := (block.Color1 & 0x001F)
	red2 := (block.Color2 & 0xF800) >> 11
	green2 := (block.Color2 & 0x07E0) >> 5
	blue2 := (block.Color2 & 0x001F)

	pixel[0].r = uint8((red1 << 3) | (red1 >> 2))
	pixel[0].g = uint8((green1 << 2) | (green1 >> 4))
	pixel[0].b = uint8((blue1 << 3) | (blue1 >> 2))

	pixel[1].r = uint8((red2 << 3) | (red2 >> 2))
	pixel[1].g = uint8((green2 << 2) | (green2 >> 4))
	pixel[1].b = uint8((blue2 << 3) | (blue2 >> 2))

	if !isDXT1 || block.Color1 > block.Color2 {
		pixel[2].r = uint8((uint16(pixel[0].r)*2 + uint16(pixel[1].r)) / 3)
		pixel[2].g = uint8((uint16(pixel[0].g)*2 + uint16(pixel[1].g)) / 3)
		pixel[2].b = uint8((uint16(pixel[0].b)*2 + uint16(pixel[1].b)) / 3)

		pixel[3].r = uint8((uint16(pixel[0].r) + uint16(pixel[1].r)*2) / 3)
		pixel[3].g = uint8((uint16(pixel[0].g) + uint16(pixel[1].g)*2) / 3)
		pixel[3].b = uint8((uint16(pixel[0].b) + uint16(pixel[1].b)*2) / 3)
		if isDXT1 {
			pixel[0].a = 0xFF
			pixel[1].a = 0xFF
			pixel[2].a = 0xFF
			pixel[3].a = 0xFF
		}
	} else {
		pixel[2].r = uint8((uint16(pixel[0].r) + uint16(pixel[1].r)) >> 1)
		pixel[2].g = uint8((uint16(pixel[0].g) + uint16(pixel[1].g)) >> 1)
		pixel[2].b = uint8((uint16(pixel[0].b) + uint16(pixel[1].b)) >> 1)

		pixel[3].r = 0
		pixel[3].g = 0
		pixel[3].b = 0

		if isDXT1 {
			pixel[0].a = 0x00
			pixel[1].a = 0x00
			pixel[2].a = 0x00
			pixel[3].a = 0x00
		}
	}

	// for _, c := range pixel {
	// 	fmt.Printf("%+v ", c)
	// }
	// fmt.Print("\n")
}

//
// helper functions
//

func (state *inflaterState) needBits(bits uint8) error {
	if bits > 32 {
		return fmt.Errorf("tried to need more than 32 bits, %v", state)
	}

	if state.bits < bits {
		if err := state.pullByte(); err != nil {
			return err
		}
	}

	return nil
}

func (state *inflaterState) pullByte() error {
	if state.bits > 32 {
		return fmt.Errorf("Tried to pull a value while we still have 32 bits available")
	}

	if (state.inputPos+1)%(0x4000) == 0 {
		state.inputPos++
	}

	var value uint32

	if state.inputPos > state.inputSize {
		if state.isEmpty {
			return fmt.Errorf("Reached end of input while trying to fetch a new byte")
		}

		state.isEmpty = true
	} else {
		value = state.input[state.inputPos]
	}

	if state.bits == 0 {
		state.head = value
		state.buffer = 0
	} else {
		state.head = state.head | (value >> state.bits)
		state.buffer = (value << (32 - state.bits))
	}

	state.bits += 32
	state.inputPos++

	return nil
}

func (state *inflaterState) dropBits(bits uint8) error {
	if bits > 32 {
		return fmt.Errorf("tried to drop more than 32 bits, %+v", state)
	}

	if bits > state.bits {
		return fmt.Errorf("tried to drop more bits than we have")
	}

	if bits == 32 {
		state.head = state.buffer
		state.buffer = 0
	} else {
		state.head <<= bits
		state.head |= (state.buffer) >> (32 - bits)
		state.buffer <<= bits
	}

	state.bits -= bits

	return nil
}

func (state *inflaterState) readBits(bits uint8) uint32 {
	return state.head >> (32 - bits)
}

func deductFormat(fourcc string) format {
	ff := format{}

	if fourcc == fccDXT1 {
		// 0
		ff.flags = ffColor | ffAlpha | ffDeducedAlphaComp
		ff.pixelSizeInBits = 4
	} else if fourcc == fccDXT5 {
		// 4
		ff.flags = ffColor | ffAlpha | ffPlainComp
		ff.pixelSizeInBits = 8
	} else {
		log.Printf("Cannot deduct format: %v", fourcc)
	}

	return ff
}
