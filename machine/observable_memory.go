package machine

import (
	"sync"

	"github.com/retronet-labs/retronet-8080/cpu"
)

// MemoryWrite descrive una scrittura richiesta e il valore effettivo finale.
type MemoryWrite struct {
	Address   uint16 `json:"address"`
	Before    byte   `json:"before"`
	Requested byte   `json:"requested"`
	After     byte   `json:"after"`
}

// MemoryWriteObserver riceve le scritture runtime, incluse quelle bloccate ROM.
type MemoryWriteObserver func(MemoryWrite)

// ObservableMemory decora cpu.Memory senza cambiare la policy del bus sottostante.
type ObservableMemory struct {
	base      cpu.Memory
	mu        sync.Mutex
	observers []MemoryWriteObserver
}

// NewObservableMemory crea un wrapper per watchpoint e trace.
func NewObservableMemory(base cpu.Memory) (*ObservableMemory, error) {
	if base == nil {
		return nil, cpu.ErrNilMemory
	}
	return &ObservableMemory{base: base}, nil
}

// Read delega al bus sottostante.
func (m *ObservableMemory) Read(addr uint16) byte {
	return m.base.Read(addr)
}

// Write delega e notifica il risultato effettivo della scrittura.
func (m *ObservableMemory) Write(addr uint16, value byte) {
	addr &= cpu.AddressMask
	before := m.base.Read(addr)
	m.base.Write(addr, value)
	write := MemoryWrite{Address: addr, Before: before, Requested: value, After: m.base.Read(addr)}

	m.mu.Lock()
	observers := append([]MemoryWriteObserver(nil), m.observers...)
	m.mu.Unlock()
	for _, observer := range observers {
		observer(write)
	}
}

// ObserveWrites aggiunge un osservatore delle scritture runtime.
func (m *ObservableMemory) ObserveWrites(observer MemoryWriteObserver) {
	if observer == nil {
		return
	}
	m.mu.Lock()
	m.observers = append(m.observers, observer)
	m.mu.Unlock()
}

// LoadBytes mantiene il caricamento privilegiato del bus senza generare eventi.
func (m *ObservableMemory) LoadBytes(addr uint16, data []byte) error {
	return LoadBytes(m.base, addr, data)
}

// LoadROM mantiene il caricamento ROM privilegiato senza generare eventi.
func (m *ObservableMemory) LoadROM(addr uint16, data []byte) error {
	if loader, ok := m.base.(romLoader); ok {
		return loader.LoadROM(addr, data)
	}
	return LoadBytes(m.base, addr, data)
}

var _ cpu.Memory = (*ObservableMemory)(nil)
