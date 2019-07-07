package main

import (
	"log"
)

const (
	maxCodeBitsLength uint32 = 32  // Max number of bits per code
	maxSymbolValue    uint32 = 285 // Max value for a symbol
	maxNbBitsHash     uint32 = 8
)

type huffmanTree struct {
	codeCompTab             [maxCodeBitsLength]uint32
	symbolValueTabOffsetTab [maxCodeBitsLength]uint16
	symbolValueTab          [maxSymbolValue]uint16
	codeBitsTab             [maxCodeBitsLength]uint8

	symbolValueHashTab [1 << maxNbBitsHash]uint16
	codeBitsHashTab    [1 << maxNbBitsHash]uint8

	isEmpty bool
}

func memset(ptr *[]uint16, value uint16, count uint32) {
	var i uint32
	for i = 0; i < count; i++ {
		(*ptr)[i] = value
	}
}

func newHuffmanTree() *huffmanTree {
	tree := huffmanTree{}

	var aWorkingBitTab [maxCodeBitsLength]uint16
	var aWorkingCodeTab [maxSymbolValue]uint16

	workingBitTab := aWorkingBitTab[:]
	workingCodeTab := aWorkingCodeTab[:]

	// Initialize our workingTabs
	memset(&workingBitTab, 0xFFFF, maxCodeBitsLength)
	memset(&workingCodeTab, 0xFFFF, maxSymbolValue)

	fillWorkingTabsHelper(1, 0x01, &workingBitTab, &workingCodeTab)

	fillWorkingTabsHelper(2, 0x12, &workingBitTab, &workingCodeTab)

	fillWorkingTabsHelper(6, 0x11, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x10, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0F, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0E, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0D, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0C, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0B, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x0A, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x09, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x08, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x07, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x06, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x05, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x04, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x03, &workingBitTab, &workingCodeTab)
	fillWorkingTabsHelper(6, 0x02, &workingBitTab, &workingCodeTab)

	// log.Printf("workingBitTab\n")
	// for _, v := range workingBitTab {
	// 	fmt.Printf("%04x\n", uint16(v))
	// }
	// log.Printf("workingCodeTab\n")
	// for _, v := range workingCodeTab {
	// 	fmt.Printf("%04x\n", uint16(v))
	// }

	tree.buildHuffmanTree(&workingBitTab, &workingCodeTab)

	// fmt.Printf("codeCompTab: ")
	// for _, v := range tree.codeCompTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Printf("\nsymbolValueTabOffsetTab: ")
	// for _, v := range tree.symbolValueTabOffsetTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Printf("\nsymbolValueTab: ")
	// for _, v := range tree.symbolValueTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Printf("\ncodeBitsTab: ")
	// for _, v := range tree.codeBitsTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Printf("\nsymbolValueHashTab: ")
	// for _, v := range tree.symbolValueHashTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Printf("\ncodeBitsHashTab: ")
	// for _, v := range tree.codeBitsHashTab {
	// 	fmt.Printf("%x ", v)
	// }

	// fmt.Println()

	return &tree
}

func (tree *huffmanTree) buildHuffmanTree(workingBitTab *[]uint16, workingCodeTab *[]uint16) {
	for i := range tree.symbolValueHashTab {
		tree.symbolValueHashTab[i] = 0xFF
	}

	var aCode uint32
	var aNbBits uint8

	// First part, filling hashTable for codes that are of less than 8 bits
	for aNbBits <= uint8(maxNbBitsHash) {
		if (*workingBitTab)[aNbBits] != 0xFFFF {
			tree.isEmpty = false

			aCurrentSymbol := (*workingBitTab)[aNbBits]
			for aCurrentSymbol != 0xFFFF {
				aHashValue := uint16(aCode << (maxNbBitsHash - uint32(aNbBits)))
				aNextHashValue := uint16((aCode + 1) << (maxNbBitsHash - uint32(aNbBits)))

				for aHashValue < aNextHashValue {
					tree.symbolValueHashTab[aHashValue] = aCurrentSymbol
					tree.codeBitsHashTab[aHashValue] = aNbBits
					aHashValue++
				}

				aCurrentSymbol = (*workingCodeTab)[aCurrentSymbol]
				aCode--
			}
		}
		aCode = (aCode << 1) + 1
		aNbBits++
	}

	var aCodeCompTabIndex uint16
	var aSymbolOffset uint16

	// Second part, filling classical structure for other codes
	for aNbBits < uint8(maxCodeBitsLength) {
		if (*workingBitTab)[aNbBits] != 0xFFFF {
			tree.isEmpty = false

			aCurrentSymbol := (*workingBitTab)[aNbBits]
			for aCurrentSymbol != 0xFFFF {
				// Registering the code
				tree.symbolValueTab[aSymbolOffset] = uint16(aCurrentSymbol)

				aSymbolOffset++
				aCurrentSymbol = (*workingCodeTab)[aCurrentSymbol]
				aCode--
			}

			// Minimum code value for aNbBits bits
			tree.codeCompTab[aCodeCompTabIndex] = (aCode + 1) << (32 - aNbBits)

			// Number of bits for l_codeCompIndex index
			tree.codeBitsTab[aCodeCompTabIndex] = aNbBits

			// Offset in symbolValueTab table to reach the value
			tree.symbolValueTabOffsetTab[aCodeCompTabIndex] = aSymbolOffset - 1

			aCodeCompTabIndex++
		}
		aCode = (aCode << 1) + 1
		aNbBits++
	}
}

func fillWorkingTabsHelper(
	iBits uint8, iSymbol uint16, workingBitTab *[]uint16, workingCodeTab *[]uint16) {

	if uint32(iBits) >= maxCodeBitsLength {
		log.Fatalf("Too many bits, got %v expected less than %v", iBits, maxCodeBitsLength)
	}
	if uint16(iSymbol) >= uint16(maxSymbolValue) {
		log.Fatalf("Too gith symbol, got %v expected less than %v", iSymbol, maxSymbolValue)
	}

	if (*workingBitTab)[iBits] == 0xFFFF {
		(*workingBitTab)[iBits] = iSymbol
	} else {
		(*workingCodeTab)[iSymbol] = (*workingBitTab)[iBits]
		(*workingBitTab)[iBits] = iSymbol
	}
}
