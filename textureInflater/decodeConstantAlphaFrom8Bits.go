package textureInflater

type constantAlpha struct {
	code  uint16
	value uint32
	alpha uint64
}

func (state *inflaterState) readConstantAlpha(aAlphaValue uint64) (alpha *constantAlpha, err error) {
	alpha = &constantAlpha{}

	if alpha.code, err = state.readCode(); err != nil {
		return
	}

	if err = state.needBits(2); err != nil {
		return
	}
	alpha.value = state.readBits(1)
	if err = state.dropBits(1); err != nil {
		return
	}

	isNotNull := uint8(state.readBits(1))
	if alpha.value != 0 {
		if err = state.dropBits(1); err != nil {
			return
		}
	}

	alpha.alpha = aAlphaValue
	if isNotNull == 0 {
		alpha.alpha = 0
	}

	return
}

func (state *inflaterState) decodeConstantAlphaFrom8Bits(ptr *[]uint8, fullFormat fullFormat) error {
	if err := state.needBits(8); err != nil {
		return err
	}
	aAlphaValueByte := uint64(state.readBits(8))
	if err := state.dropBits(8); err != nil {
		return err
	}

	var aPixelBlockPos uint32

	aAlphaValue := aAlphaValueByte | uint64(aAlphaValueByte<<8)

	for aPixelBlockPos < fullFormat.nbObPixelBlocks {
		alpha, err := state.readConstantAlpha(aAlphaValue)
		if err != nil {
			return err
		}

		aPixelBlockPos = state.applyConstantAlphaFrom8Bits(aPixelBlockPos, fullFormat, *alpha, ptr)

		for aPixelBlockPos < fullFormat.nbObPixelBlocks && state.alphaBitMap[aPixelBlockPos] {
			aPixelBlockPos++
		}
	}

	return nil
}

func (state *inflaterState) applyConstantAlphaFrom8Bits(aPixelBlockPos uint32, fullFormat fullFormat, alpha constantAlpha, ptr *[]uint8) uint32 {
	ioOutputTab := *ptr
	for alpha.code > 0 {
		if !state.alphaBitMap[aPixelBlockPos] {
			if alpha.value != 0 {
				offset := fullFormat.bytesPerPixelBlock * aPixelBlockPos

				//fmt.Printf("%016x %v\n", value, fullFormat.bytesPerComponent)

				var i uint32
				for i = 0; i < fullFormat.bytesPerComponent; i++ {
					ioOutputTab[offset+i] = uint8((alpha.alpha >> (8 * i)) & 0xFF)
				}

				state.alphaBitMap[aPixelBlockPos] = true
			}
			alpha.code--
		}
		aPixelBlockPos++

	}
	return aPixelBlockPos
}
