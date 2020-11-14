package textureInflater

func (state *inflaterState) decodeWhiteColor(ptr *[]uint8, fullFormat fullFormat) error {
	var aPixelBlockPos uint32

	for aPixelBlockPos < fullFormat.nbObPixelBlocks {
		aCode, err := state.readCode()
		if err != nil {
			return err
		}

		if err := state.needBits(1); err != nil {
			return err
		}
		aValue := state.readBits(1)
		if err := state.dropBits(1); err != nil {
			return err
		}

		aPixelBlockPos = state.applyWhiteColor(aCode, aPixelBlockPos, aValue, fullFormat, ptr)

		for aPixelBlockPos < fullFormat.nbObPixelBlocks && state.colorBitMap[aPixelBlockPos] {
			aPixelBlockPos++
		}
	}
	return nil
}

func (state *inflaterState) applyWhiteColor(aCode uint16, aPixelBlockPos uint32, aValue uint32, fullFormat fullFormat, ptr *[]uint8) uint32 {
	ioOutputTab := *ptr

	for aCode > 0 {
		if !state.alphaBitMap[aPixelBlockPos] {
			if aValue != 0 {
				offset := fullFormat.bytesPerPixelBlock * aPixelBlockPos

				var i uint32
				for i = 0; i < fullFormat.bytesPerComponent; i++ {
					ioOutputTab[offset+i] = 0xFF
				}

				state.alphaBitMap[aPixelBlockPos] = true
				state.colorBitMap[aPixelBlockPos] = true
			}
			aCode--
		}
		aPixelBlockPos++

	}
	return aPixelBlockPos
}
