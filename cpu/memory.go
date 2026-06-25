package cpu

const AddressSpaceSize = int(AddressMask) + 1

// Memory e' il bus memoria visto dal core 8080.
type Memory interface {
	Read(addr uint16) byte
	Write(addr uint16, value byte)
}

// FlatMemory modella lo spazio lineare da 64 KB dell'Intel 8080.
type FlatMemory struct {
	Data [AddressSpaceSize]byte
}

func NewFlatMemory() *FlatMemory { return &FlatMemory{} }

func (m *FlatMemory) Read(addr uint16) byte { return m.Data[addr] }

func (m *FlatMemory) Write(addr uint16, value byte) { m.Data[addr] = value }
