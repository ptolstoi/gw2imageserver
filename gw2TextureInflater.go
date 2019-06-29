package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

func inflate(inputRaw []byte, outputSize uint32) (*[]bgra, error) {
	input := make([]uint32, len(inputRaw)/4)
	binary.Read(bytes.NewBuffer(inputRaw[:]), binary.LittleEndian, &input)

	state := inflaterState{
		input:     input,
		inputSize: uint32(len(input) / 4),
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

	if formatFourCC == fccDXT1n {
		log.Printf("fccDXT1")
		colors, err := processDXT1(&result, aFullFormat.width, aFullFormat.height)

		log.Printf("colors: %v, error: %v", len(colors), err)
	} else if formatFourCC == fccDXT5n {
		log.Printf("fccDXT5")
	} else {
		log.Printf("unknown formatFourCC: %08X", formatFourCC)
	}

	log.Printf("size: %v", len(result))

	return nil, nil
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
		log.Printf("LOOP2")
		aColorSize := uint32(len(aColorBitmap))
		for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
			if aColorBitmap[i] {
				offset := fullFormat.bytesPerPixelBlock * i
				if fullFormat.hasTwoComponents {
					offset += fullFormat.bytesPerComponent
				}

				data := state.input[state.inputPos]

				ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
				ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
				ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
				ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

				state.inputPos++
			}
		}

		if fullFormat.bytesPerComponent > 4 {
			log.Printf("LOOP2")

			// for (aLoopIndex = 0; aLoopIndex < aColorBitmap.size() && iState.inputPos < iState.inputSize; ++aLoopIndex) {
			// 	if (!aColorBitmap[aLoopIndex]) {
			// 		uint32_t aOffset = iFullFormat.bytesPerPixelBlock * aLoopIndex + 4 + (iFullFormat.hasTwoComponents ? iFullFormat.bytesPerComponent : 0);
			// 		(*reinterpret_cast<uint32_t*>(&(ioOutputTab[aOffset]))) = iState.input[iState.inputPos];
			// 		++iState.inputPos;
			// 	}
			// }
			for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
				offset := fullFormat.bytesPerPixelBlock*i + 4
				if fullFormat.hasTwoComponents {
					offset += fullFormat.bytesPerComponent
				}

				data := state.input[state.inputPos]

				ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
				ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
				ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
				ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

				state.inputPos++
			}
		}
	}

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

func processDXT1(data *[]uint8, width uint16, height uint16) ([]bgra, error) {
	/*
		reinterpret color:
			union DXTColor {
				struct {
					uint16 red1 : 5;
					uint16 green1 : 6;
					uint16 blue1 : 5;
					uint16 red2 : 5;
					uint16 green2 : 6;
					uint16 blue2 : 5;
				};
				struct {
					uint16 color1;
					uint16 color2;
				};
			};

			struct DXT1Block {
				DXTColor colors;
				uint32   indices;
			};
	*/

	// numPixels := width * height

	type DXT1Block struct {
		Color1  uint16
		Color2  uint16
		Indices uint32
	}

	blocks := make([]DXT1Block, len(*data)/8)

	reader := bytes.NewBuffer(*data)
	if err := binary.Read(reader, binary.LittleEndian, &blocks); err != nil {
		return nil, err
	}

	// pixels := make([]bgra, numPixels)

	numHorizBlocks := width >> 2
	numVertBlocks := width >> 2

	var y uint16
	var x uint16

	for y = 0; y < numVertBlocks; y++ {
		for x = 0; x < numHorizBlocks; x++ {
			// block := blocks[y*numHorizBlocks+x]
			// processDXT1Block(&pixels, block, x * 4, y * 4, width)
		}
	}

	return nil, nil
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
