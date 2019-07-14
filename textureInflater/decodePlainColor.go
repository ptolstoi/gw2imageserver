package textureInflater

type doubleColor struct {
	red1   uint8
	red2   uint8
	green1 uint16
	green2 uint8
	blue1  uint8
	blue2  uint8
}

func (state *inflaterState) decodePlainColor(ptr *[]uint8, fullFormat fullFormat) error {
	aRed, aBlue, aGreen, err := state.readPlainColor()
	if err != nil {
		return err
	}

	//log.Printf("[decodePlainColor] frst: b: %04x g: %04x r: %04x", aBlue, aGreen, aRed)

	temp := extractTempColors(aRed, aBlue, aGreen)

	//log.Printf("[decodePlainColor] tmp2: b: %04x g: %04x r: %04x", aBlueTemp2, aGreenTemp2, aRedTemp2)

	aRedFlg, aBlueFlg, aGreenFlg := extractColorFlags(temp)

	aCompRed := 12 * (uint32(aRed) - uint32(temp.red2)) / (8 - aRedFlg)
	aCompBlue := 12 * (uint32(aBlue) - uint32(temp.blue2)) / (8 - aBlueFlg)

	aCompGreen := 12 * (uint32(aGreen) - uint32(temp.green2)) / (8 - aGreenFlg)

	//log.Printf("[decodePlainColor] tmp2: b: %04x g: %04x r: %04x", aCompBlue, aCompGreen, aCompRed)

	aValueRed1, aValueRed2, aValueBlue1, aValueBlue2, aValueGreen1, aValueGreen2 := state.magicSplitColors(aCompRed, aCompBlue, aCompGreen, temp)

	//log.Printf("[decodePlainColor] red : 1: %04x 2: %04x", aValueRed1, aValueRed2)
	//log.Printf("[decodePlainColor] blue: 1: %04x 2: %04x", aValueBlue1, aValueBlue2)
	//log.Printf("[decodePlainColor] gren: 1: %04x 2: %04x", aValueGreen1, aValueGreen2)

	aValueColor1 := uint32(aValueRed1) | ((aValueGreen1 | (aValueBlue1 << 6)) << 5)
	aValueColor2 := uint32(aValueRed2) | ((aValueGreen2 | (aValueBlue2 << 6)) << 5)

	//log.Printf("[decodePlainColor] finl: 1: %04x 2: %04x", aValueColor1, aValueColor2)

	var aTempValue1 uint32
	var aTempValue2 uint32

	aTempValue1, aTempValue2 = magicValueSplit2(
		aTempValue1, aTempValue2,
		aValueRed1, aValueRed2,
		uint16(temp.red1), aCompRed,
	)

	//log.Printf("[decodePlainColor] red : 1: %04x 2: %04x", aTempValue1, aTempValue2)

	aTempValue1, aTempValue2 = magicValueSplit2(
		aTempValue1, aTempValue2,
		aValueBlue1, aValueBlue2,
		uint16(temp.blue1), aCompBlue,
	)

	//log.Printf("[decodePlainColor] blue: 1: %04x 2: %04x", aTempValue1, aTempValue2)

	aTempValue1, aTempValue2 = magicValueSplit2(
		aTempValue1, aTempValue2,
		aValueGreen1, aValueGreen2,
		uint16(temp.green1), aCompGreen,
	)

	//log.Printf("[decodePlainColor] gren: 1: %04x 2: %04x", aTempValue1, aTempValue2)

	if aTempValue2 > 0 {
		aTempValue1 = (aTempValue1 + (aTempValue2 / 2)) / aTempValue2
	}

	//log.Printf("[decodePlainColor] temp: 1: %04x 2: %04x", aTempValue1, aTempValue2)

	aDxt1SpecialCase := ((fullFormat.flags & ffDeducedAlphaComp) != 0) && (aTempValue1 == 5 || aTempValue1 == 6 || aTempValue2 != 0)

	//log.Printf("[decodePlainColor] aDxt1SpecialCase: %v", aDxt1SpecialCase)

	if aTempValue2 > 0 && !aDxt1SpecialCase {
		if aValueColor2 == 0xFFFF {
			aTempValue1 = 12
			aValueColor1--
		} else {
			aTempValue1 = 0
			aValueColor2++
		}
	}

	//log.Printf("[decodePlainColor] sptl: 1: %04x 2: %04x", aTempValue1, aTempValue2)

	if aValueColor2 >= aValueColor1 {
		aValueColor1, aValueColor2 = aValueColor2, aValueColor1

		aTempValue1 = 12 - aTempValue1
	}

	//log.Printf("[decodePlainColor] sptl: 1: %04x 2: %04x", aTempValue1, aTempValue2)

	var aColorChosen uint64

	if aDxt1SpecialCase {
		aColorChosen = 2
	} else {
		if aTempValue1 < 2 {
			aColorChosen = 0
		} else if aTempValue1 < 6 {
			aColorChosen = 2
		} else if aTempValue1 < 10 {
			aColorChosen = 3
		} else {
			aColorChosen = 1
		}
	}

	//log.Printf("[decodePlainColor] chosen: %04x", aColorChosen)

	aTempValue := (aColorChosen) | (aColorChosen << 2) | ((aColorChosen | (aColorChosen << 2)) << 4)
	aTempValue = aTempValue | (aTempValue << 8)
	aTempValue = aTempValue | (aTempValue << 16)

	//log.Printf("[decodePlainColor] aTempValue: %04x", aTempValue)

	aFinalValue := uint64(aValueColor1) | uint64(aValueColor2<<16) | (uint64(aTempValue) << 32)

	//log.Printf("[decodePlainColor] aFinalValue: %04x", aFinalValue)

	return state.extractAndApplyPlainColor(aFinalValue, ptr, fullFormat)
}

func extractTempColors(aRed uint16, aBlue uint16, aGreen uint16) doubleColor {
	temp := doubleColor{
		red1:  uint8((aRed - (aRed >> 5)) >> 3),
		blue1: uint8((aBlue - (aBlue >> 5)) >> 3),

		green1: uint16((aGreen - (aGreen >> 6)) >> 2),
	}
	//log.Printf("[decodePlainColor] tmp1: b: %04x g: %04x r: %04x", aBlueTemp1, aGreenTemp1, aRedTemp1)
	temp.red2 = uint8((temp.red1 << 3) + (temp.red1 >> 2))
	temp.blue2 = uint8((temp.blue1 << 3) + (temp.blue1 >> 2))
	temp.green2 = uint8((temp.green1 << 2) + (temp.green1 >> 4))
	return temp
}

func (state *inflaterState) magicSplitColors(aCompRed uint32, aCompBlue uint32, aCompGreen uint32, temp doubleColor) (uint32, uint32, uint32, uint32, uint32, uint32) {
	aValueRed1, aValueRed2 := magicValueSplit(aCompRed, uint32(temp.red1))
	aValueBlue1, aValueBlue2 := magicValueSplit(aCompBlue, uint32(temp.blue1))
	aValueGreen1, aValueGreen2 := magicValueSplit(aCompGreen, uint32(temp.green1))

	return aValueRed1, aValueRed2, aValueBlue1, aValueBlue2, aValueGreen1, aValueGreen2
}

func extractColorFlags(temp doubleColor) (uint32, uint32, uint32) {
	aRedFlg := uint32(0)
	aBlueFlg := uint32(0)
	aGreenFlg := uint32(0)
	if temp.red1&0x11 == 0x11 {
		aRedFlg = 1
	}
	if temp.blue1&0x11 == 0x11 {
		aBlueFlg = 1
	}
	if temp.green1&0x1111 == 0x1111 {
		aGreenFlg = 1
	}
	return aRedFlg, aBlueFlg, aGreenFlg
}

func (state *inflaterState) readPlainColor() (aBlue uint16, aRed uint16, aGreen uint16, err error) {
	if err = state.needBits(24); err != nil {
		return
	}

	aBlue = uint16(state.readBits(8))
	if err = state.dropBits(8); err != nil {
		return
	}
	aGreen = uint16(state.readBits(8))
	if err = state.dropBits(8); err != nil {
		return
	}
	aRed = uint16(state.readBits(8))
	if err = state.dropBits(8); err != nil {
		return
	}

	return
}

func (state *inflaterState) extractAndApplyPlainColor(aFinalValue uint64, ptr *[]uint8, fullFormat fullFormat) error {
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

		// log.Printf("%04x %04x %04x", aCode, aValue, aPixelBlockPos)

		aPixelBlockPos = state.applyPlainColor(aCode, aPixelBlockPos, aValue, fullFormat, ptr, aFinalValue)

		for aPixelBlockPos < fullFormat.nbObPixelBlocks && state.colorBitMap[aPixelBlockPos] {
			aPixelBlockPos++
		}
	}

	return nil
}

func (state *inflaterState) applyPlainColor(aCode uint16, aPixelBlockPos uint32, aValue uint32, fullFormat fullFormat, ptr *[]uint8, aFinalValue uint64) uint32 {
	ioOutputTab := *ptr

	for aCode > 0 {
		if state.colorBitMap[aPixelBlockPos] {
			aPixelBlockPos++
			continue
		}

		if aValue != 0 {
			offset := fullFormat.bytesPerPixelBlock * aPixelBlockPos
			if fullFormat.hasTwoComponents {
				offset += fullFormat.bytesPerComponent
			}

			var i uint32
			for i = 0; i < fullFormat.bytesPerComponent; i++ {
				ioOutputTab[offset+i] = uint8((aFinalValue >> (8 * i)) & 0xFF)
			}

			state.colorBitMap[aPixelBlockPos] = true
		}
		aCode--

		aPixelBlockPos++
	}
	return aPixelBlockPos
}

func magicValueSplit(aComp uint32, aTemp1 uint32) (uint32, uint32) {
	var aValue1 uint32
	var aValue2 uint32

	if aComp < 2 {
		aValue1 = aTemp1
		aValue2 = aTemp1
	} else if aComp < 6 {
		aValue1 = aTemp1
		aValue2 = aTemp1 + 1
	} else if aComp < 10 {
		aValue1 = aTemp1 + 1
		aValue2 = aTemp1
	} else {
		aValue1 = aTemp1 + 1
		aValue2 = aTemp1 + 1
	}

	return aValue1, aValue2
}

func magicValueSplit2(
	aTempValue1 uint32, aTempValue2 uint32,
	aValueRed1 uint32, aValueRed2 uint32,
	aRedTemp1 uint16, aCompRed uint32) (uint32, uint32) {
	if aValueRed1 != aValueRed2 {
		if aValueRed1 == uint32(aRedTemp1) {
			aTempValue1 += aCompRed
		} else {
			aTempValue1 += 12 - aCompRed
		}
		aTempValue2++
	}

	return aTempValue1, aTempValue2
}
