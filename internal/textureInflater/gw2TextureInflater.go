package textureInflater

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	imageColor "image/color"
	"log"

	"github.com/ptolstoi/gw2imageserver/internal/huffman"
)

const (
	FccDXT1 = "\x44\x58\x54\x31"

	FccDXT5 = "\x44\x58\x54\x35"

	fccDXT1n uint32 = 0x31545844
	fccDXT5n uint32 = 0x35545844
)

type inflaterState struct {
	input     []uint32
	inputSize uint32
	inputPos  uint32

	head   uint32
	buffer uint32
	bits   uint8

	isEmpty bool

	colorBitMap []bool
	alphaBitMap []bool

	huffmanTree huffman.HuffmanTree
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
	fourCC uint32
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

func newInflaterState(input *[]uint32) *inflaterState {
	tree := huffman.NewHuffmanTree()

	state := inflaterState{
		input:     *input,
		inputSize: uint32(len(*input)),
		inputPos:  0,

		head:   0,
		bits:   0,
		buffer: 0,

		isEmpty: false,

		huffmanTree: tree,
	}
	return &state
}

func Inflate(inputRaw []byte, origWidth uint16, origHeight uint16) (image.Image, error) {
	input := make([]uint32, len(inputRaw)/4)
	if err := binary.Read(bytes.NewBuffer(inputRaw[:]), binary.LittleEndian, &input); err != nil {
		return nil, err
	}

	state := newInflaterState(&input)

	aFullFormat, err := state.readFullFormat()
	if err != nil {
		return nil, err
	}

	//log.Printf("FullFormat: {flags:% 016b(%[1]v) pixelSizeInBits:%v} %+v",
	//	aFullFormat.flags,
	//	aFullFormat.pixelSizeInBits,
	//	aFullFormat)

	anOutputSize := aFullFormat.bytesPerPixelBlock * aFullFormat.nbObPixelBlocks

	// log.Printf("anOutputSize: %v", anOutputSize)

	result, err := state.inflateData(*aFullFormat, anOutputSize)
	if err != nil {
		return nil, err
	}

	//log.Printf("afterInflate: inputPos=%v len(input)=%v inputSize=%v", state.inputPos, len(state.input), state.inputSize)

	var colors *[]bgra
	if aFullFormat.fourCC == fccDXT1n {
		// log.Printf("fccDXT1")
		var err error
		colors, err = processDXT1(&result, aFullFormat.width, aFullFormat.height)

		if err != nil {
			return nil, err
		}

		// log.Printf("colors: %v", len(*colors))
	} else if aFullFormat.fourCC == fccDXT5n {
		// log.Printf("fccDXT5")

		var err error
		colors, err = processDXT5(&result, aFullFormat.width, aFullFormat.height)
		if err != nil {
			return nil, err
		}
	} else {
		f := aFullFormat.fourCC
		c := func(a uint32) string {
			return string(rune(a))
		}
		return nil, fmt.Errorf("unknown formatFourCC: 0x%08x (%v%v%v%v)", aFullFormat.fourCC,
			c((f>>0)&0xFF), c((f>>8)&0xFF), c((f>>16)&0xFF), c((f>>24)&0xFF),
		)
	}

	img := image.NewNRGBA(image.Rect(0, 0, int(origWidth), int(origHeight)))
	var y uint16
	var x uint16
	for y = 0; y < origHeight; y++ {
		for x = 0; x < origWidth; x++ {
			index := uint(y)*uint(origWidth) + uint(x)
			color := (*colors)[index]

			//fmt.Printf("%04x %04x: %02x%02x%02x%02x\n",
			//	x, y,
			//	color.r, color.g, color.b, color.a,
			//	)

			img.SetNRGBA(int(x), int(y), imageColor.NRGBA{
				R: color.r,
				G: color.g,
				B: color.b,
				A: color.a,
			})
		}
	}

	return img, nil
}

func (state *inflaterState) inflateData(fullFormat fullFormat, outputSize uint32) ([]uint8, error) {
	ioOutputTab := make([]uint8, outputSize)

	state.head = 0
	state.bits = 0
	state.buffer = 0

	if err := state.needBits(32); err != nil {
		return nil, err
	}
	// aDataSize := state.readBits(32)
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	if err := state.needBits(32); err != nil {
		return nil, err
	}
	aCompressionFlags := state.readBits(32)
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	state.colorBitMap = make([]bool, fullFormat.nbObPixelBlocks)
	state.alphaBitMap = make([]bool, fullFormat.nbObPixelBlocks)

	// log.Printf("aDataSize: %v, aCompressionFlags: %032b", aDataSize, aCompressionFlags)

	if err := state.decompress(aCompressionFlags, &ioOutputTab, fullFormat); err != nil {
		return nil, err
	}

	if state.bits >= 32 {
		state.inputPos--
	}

	if err := state.processAlpha(&ioOutputTab, fullFormat); err != nil {
		return nil, err
	}

	if err := state.processColor(&ioOutputTab, fullFormat); err != nil {
		return nil, err
	}

	//var i uint32
	//size := uint32(len(ioOutputTab))
	//for i = 0; i < size; i++ {
	//	fmt.Printf("%02x", ioOutputTab[i])
	//	if (i+1)%uint32(fullFormat.width) == 0 {
	//		fmt.Printf("\n")
	//	}
	//}
	//fmt.Printf("\n")

	return ioOutputTab, nil
}

func (state *inflaterState) processAlpha(ptr *[]uint8, fullFormat fullFormat) error {
	ioOutputTab := *ptr

	if ((fullFormat.flags&ffAlpha) != 0 && (fullFormat.flags&ffDeducedAlphaComp) == 0) || (fullFormat.flags&ffBiColorComp) != 0 {
		//log.Printf("LOOP1")

		var i uint32

		alphaBitMapSize := uint32(len(state.alphaBitMap))

		for i = 0; i < alphaBitMapSize && state.inputPos < state.inputSize; i++ {
			if !state.alphaBitMap[i] {
				offset := fullFormat.bytesPerPixelBlock * i

				data := state.input[state.inputPos]

				// fmt.Printf("%08X\n", data)

				ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
				ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
				ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
				ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

				state.inputPos++

				if fullFormat.bytesPerComponent > 4 {
					offset += 4

					data := state.input[state.inputPos]

					ioOutputTab[offset+0] = uint8((data >> 0) & 0xFF)
					ioOutputTab[offset+1] = uint8((data >> 8) & 0xFF)
					ioOutputTab[offset+2] = uint8((data >> 16) & 0xFF)
					ioOutputTab[offset+3] = uint8((data >> 24) & 0xFF)

					state.inputPos++
				}
			}
		}
	}

	return nil
}

func (state *inflaterState) processColor(ptr *[]uint8, fullFormat fullFormat) error {
	ioOutputTab := *ptr

	if (fullFormat.flags&ffColor) != 0 || (fullFormat.flags&ffBiColorComp) != 0 {
		aColorSize := uint32(len(state.colorBitMap))
		//log.Printf("LOOP2 %v %v %v", aColorSize, state.inputPos, state.inputSize)

		var i uint32
		for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
			if !state.colorBitMap[i] {
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
			state.processColorMultiByte(ptr, fullFormat)
		}
	}

	return nil
}

func (state *inflaterState) processColorMultiByte(ptr *[]uint8, fullFormat fullFormat) {
	ioOutputTab := *ptr
	aColorSize := uint32(len(state.colorBitMap))

	var i uint32
	for i = 0; i < aColorSize && state.inputPos < state.inputSize; i++ {
		if !state.colorBitMap[i] {
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

func processDXTColor(pixel *[4]bgra, block *dxtColor, setAlpha bool, isDXT1 bool) {
	// fmt.Printf("% 5x % 5x\n", block.color1, block.color2)

	red1 := uint8((block.color1 & 0xF800) >> 11)
	green1 := uint8((block.color1 & 0x07E0) >> 5)
	blue1 := uint8(block.color1 & 0x001F)
	red2 := uint8((block.color2 & 0xF800) >> 11)
	green2 := uint8((block.color2 & 0x07E0) >> 5)
	blue2 := uint8(block.color2 & 0x001F)

	pixel[0].r = uint8((red1 << 3) | (red1 >> 2))
	pixel[0].g = uint8((green1 << 2) | (green1 >> 4))
	pixel[0].b = uint8((blue1 << 3) | (blue1 >> 2))

	pixel[1].r = uint8((red2 << 3) | (red2 >> 2))
	pixel[1].g = uint8((green2 << 2) | (green2 >> 4))
	pixel[1].b = uint8((blue2 << 3) | (blue2 >> 2))

	if !isDXT1 || block.color1 > block.color2 {
		pixel[2].r = uint8((uint16(pixel[0].r)*2 + uint16(pixel[1].r)) / 3)
		pixel[2].g = uint8((uint16(pixel[0].g)*2 + uint16(pixel[1].g)) / 3)
		pixel[2].b = uint8((uint16(pixel[0].b)*2 + uint16(pixel[1].b)) / 3)

		pixel[3].r = uint8((uint16(pixel[0].r) + uint16(pixel[1].r)*2) / 3)
		pixel[3].g = uint8((uint16(pixel[0].g) + uint16(pixel[1].g)*2) / 3)
		pixel[3].b = uint8((uint16(pixel[0].b) + uint16(pixel[1].b)*2) / 3)

		if setAlpha {
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

		if setAlpha {
			pixel[0].a = 0xFF
			pixel[1].a = 0xFF
			pixel[2].a = 0xFF
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
	if state.bits >= 32 {
		return fmt.Errorf("Tried to pull a value while we still have 32 bits available")
	}

	if (state.inputPos+1)%(0x4000) == 0 {
		state.inputPos++
	}

	var value uint32

	if state.inputPos >= state.inputSize {
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
		state.buffer = value << (32 - state.bits)
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

func (state *inflaterState) readCode() (ioCode uint16, err error) {
	if state.huffmanTree.IsEmpty() {
		return 0, fmt.Errorf("huffmanTree not initialized")
	}

	if err := state.needBits(32); err != nil {
		return 0, err
	}

	symbol := state.readBits(uint8(huffman.MaxNbBitsHash))

	if state.huffmanTree.GetSymbolValueHash(symbol) != 0xFFFF {
		ioCode = uint16(state.huffmanTree.GetSymbolValueHash(symbol))
		bitsToDrop := state.huffmanTree.GetCodeBitsHash(symbol)
		if err := state.dropBits(bitsToDrop); err != nil {
			return 0, err
		}
	} else {
		var anIndex uint32
		tmp := state.readBits(32)
		for tmp < state.huffmanTree.GetCodeComp(anIndex) {
			anIndex++
		}

		aNbBits := state.huffmanTree.GetCodeBits(anIndex)
		symbol = uint32(state.huffmanTree.GetSymbolValueHash(anIndex)) -
			((tmp - state.huffmanTree.GetCodeComp(anIndex)) >> (32 - aNbBits))

		ioCode = state.huffmanTree.GetSymbolValue(symbol)
		if err := state.dropBits(aNbBits); err != nil {
			return 0, err
		}
	}

	return
}

func (state *inflaterState) decompress(aCompressionFlags uint32, ioOutputTab *[]uint8, fullFormat fullFormat) error {
	if aCompressionFlags&cfDecodeWhiteColor != 0 {
		//log.Printf("cfDecodeWhiteColor")
		// return nil, fmt.Errorf("cfDecodeWhiteColor not implemented")
		if err := state.decodeWhiteColor(ioOutputTab, fullFormat); err != nil {
			return err
		}
	}
	if aCompressionFlags&cfDecodeConstantAlphaFrom4Bits != 0 {
		// log.Printf("cfDecodeConstantAlphaFrom4Bits")
		return fmt.Errorf("cfDecodeConstantAlphaFrom4Bits not implemented")
	}
	if aCompressionFlags&cfDecodeConstantAlphaFrom8Bits != 0 {
		//log.Printf("cfDecodeConstantAlphaFrom8Bits")
		// return fmt.Errorf("cfDecodeConstantAlphaFrom8Bits not implemented")
		if err := state.decodeConstantAlphaFrom8Bits(ioOutputTab, fullFormat); err != nil {
			return err
		}
	}
	if aCompressionFlags&cfDecodePlainColor != 0 {
		//log.Printf("cfDecodePlainColor")
		// return fmt.Errorf("cfDecodePlainColor not implemented")
		if err := state.decodePlainColor(ioOutputTab, fullFormat); err != nil {
			return err
		}
	}

	return nil
}

func (state *inflaterState) readFullFormat() (*fullFormat, error) {
	// skip header
	// log.Printf("skipping header")
	if err := state.needBits(32); err != nil {
		return nil, err
	}
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	// log.Printf("reading format")
	// format
	if err := state.needBits(32); err != nil {
		return nil, err
	}
	formatFourCC := state.readBits(32)
	if err := state.dropBits(32); err != nil {
		return nil, err
	}

	//log.Printf("fourmatFourCC: % x", formatFourCC)

	format := deductFormat(string(rune(formatFourCC&0xFF)) +
		string(rune(formatFourCC&0xFF00>>8)) +
		string(rune(formatFourCC&0xFF0000>>16)) +
		string(rune(formatFourCC&0xFF000000>>24)))
	aFullFormat := fullFormat{
		format: &format,
		fourCC: formatFourCC,
	}

	// log.Printf("reading size")
	if err := state.needBits(32); err != nil {
		return nil, err
	}
	aFullFormat.height = uint16(state.readBits(16))
	//log.Printf("width=%v", aFullFormat.width)
	if err := state.dropBits(16); err != nil {
		return nil, err
	}
	aFullFormat.width = uint16(state.readBits(16))
	//log.Printf("height=%v", aFullFormat.height)
	if err := state.dropBits(16); err != nil {
		return nil, err
	}

	aFullFormat.nbObPixelBlocks = uint32((aFullFormat.width+3)/4) * uint32((aFullFormat.height+3)/4)
	aFullFormat.bytesPerPixelBlock = uint32(aFullFormat.pixelSizeInBits) * 4 * 4 / 8
	aFullFormat.hasTwoComponents = ((aFullFormat.flags & (ffPlainComp | ffColor | ffAlpha)) == (ffPlainComp | ffColor | ffAlpha)) || (aFullFormat.flags&ffBiColorComp) != 0
	aFullFormat.bytesPerComponent = aFullFormat.bytesPerPixelBlock
	if aFullFormat.hasTwoComponents {
		aFullFormat.bytesPerComponent = aFullFormat.bytesPerPixelBlock / 2
	}

	return &aFullFormat, nil
}

func deductFormat(fourcc string) format {
	ff := format{}

	if fourcc == FccDXT1 {
		// 0
		ff.flags = ffColor | ffAlpha | ffDeducedAlphaComp
		ff.pixelSizeInBits = 4
	} else if fourcc == FccDXT5 {
		// 4
		ff.flags = ffColor | ffAlpha | ffPlainComp
		ff.pixelSizeInBits = 8
	} else {
		log.Printf("[deductFormat] Cannot deduct format: %v", fourcc)
	}

	return ff
}
