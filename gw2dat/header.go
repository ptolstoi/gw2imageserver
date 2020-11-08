package gw2dat

type GW2DatHeader struct {
	Version       uint8
	Identifier    [3]uint8
	HeaderSize    uint32
	UnknownField1 uint32
	ChunkSize     uint32
	CRC           uint32
	UnknownField2 uint32
	MFTOffset     uint64
	MFTSize       uint32
	Flags         uint32
}

type MFTHeader struct {
	Magic           [4]uint8
	UnknownField1   uint32
	UnknownField2   uint32
	NumberOfEntries uint32
	UnknownField3   uint64
}

type MFTEntry struct {
	Offset           uint64
	Size             uint32
	CompressionFlags uint16
	UnknownField1    uint16
	UnknownField2    uint32
	Crc              uint32
}
