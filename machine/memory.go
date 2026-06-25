package machine

import (
	"errors"
	"fmt"

	"github.com/retronet-labs/retronet-8080/cpu"
)

var (
	ErrReadOnlyMemory = errors.New("memoria in sola lettura")
	ErrUnmappedMemory = errors.New("memoria non mappata")
)

// MemoryBus applica una mappa ROM/RAM allo spazio indirizzi a 16 bit.
// Gli accessi CPU a regioni ROM ignorano le scritture; le regioni non mappate
// restituiscono 0xFF, come valore convenzionale di open bus.
type MemoryBus struct {
	data   [cpu.AddressSpaceSize]byte
	kinds  [cpu.AddressSpaceSize]MemoryKind
	mapped [cpu.AddressSpaceSize]bool
}

// NewMemoryBus crea un bus dalle regioni fornite e rifiuta mappe ambigue.
func NewMemoryBus(regions []MemoryRegion) (*MemoryBus, error) {
	bus := &MemoryBus{}
	for _, region := range regions {
		if err := validateMemoryRegion(region); err != nil {
			return nil, err
		}
		for addr := int(region.Start); addr <= int(region.End); addr++ {
			if bus.mapped[addr] {
				return nil, fmt.Errorf("regione memoria %q sovrapposta a 0x%04X", region.Name, addr)
			}
			bus.mapped[addr] = true
			bus.kinds[addr] = region.Kind
		}
	}
	return bus, nil
}

func validateMemoryRegion(region MemoryRegion) error {
	if region.Start > cpu.AddressMask || region.End > cpu.AddressMask {
		return fmt.Errorf("regione memoria %q fuori dallo spazio 16 bit", region.Name)
	}
	if region.Start > region.End {
		return fmt.Errorf("regione memoria %q: inizio 0x%04X dopo fine 0x%04X", region.Name, region.Start, region.End)
	}
	switch region.Kind {
	case MemoryKindRAM, MemoryKindROM, MemoryKindMixed:
		return nil
	default:
		return fmt.Errorf("regione memoria %q: tipo %q non valido", region.Name, region.Kind)
	}
}

// Read legge un byte, mascherando l'indirizzo come il bus fisico dell'8080.
func (m *MemoryBus) Read(addr uint16) byte {
	index := int(addr & cpu.AddressMask)
	if !m.mapped[index] {
		return 0xFF
	}
	return m.data[index]
}

// Write scrive solo nelle regioni RAM o mixed. Le scritture ROM e open bus
// vengono ignorate per rispettare il contratto cpu.Memory, che non espone errori.
func (m *MemoryBus) Write(addr uint16, value byte) {
	index := int(addr & cpu.AddressMask)
	if !m.mapped[index] || m.kinds[index] == MemoryKindROM {
		return
	}
	m.data[index] = value
}

// Kind restituisce il tipo effettivo della cella dopo eventuali caricamenti ROM.
func (m *MemoryBus) Kind(addr uint16) (MemoryKind, bool) {
	index := int(addr & cpu.AddressMask)
	return m.kinds[index], m.mapped[index]
}

// LoadBytes inizializza RAM o memoria mixed senza aggirare regioni ROM.
func (m *MemoryBus) LoadBytes(addr uint16, data []byte) error {
	if err := ValidateRange(addr, len(data)); err != nil {
		return err
	}
	for i := range data {
		index := int(addr) + i
		if !m.mapped[index] {
			return fmt.Errorf("caricamento a 0x%04X: %w", index, ErrUnmappedMemory)
		}
		if m.kinds[index] == MemoryKindROM {
			return fmt.Errorf("caricamento a 0x%04X: %w", index, ErrReadOnlyMemory)
		}
	}
	copy(m.data[int(addr):int(addr)+len(data)], data)
	return nil
}

// LoadROM inizializza privilegiatamente una ROM e protegge l'intervallo caricato.
func (m *MemoryBus) LoadROM(addr uint16, data []byte) error {
	if err := ValidateRange(addr, len(data)); err != nil {
		return err
	}
	for i := range data {
		index := int(addr) + i
		if !m.mapped[index] {
			return fmt.Errorf("ROM a 0x%04X: %w", index, ErrUnmappedMemory)
		}
	}
	for i, value := range data {
		index := int(addr) + i
		m.data[index] = value
		m.kinds[index] = MemoryKindROM
	}
	return nil
}

var _ cpu.Memory = (*MemoryBus)(nil)
