package machine

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/retronet-labs/retronet-8080/cpu"
)

var (
	ErrPeripheralNameRequired = errors.New("nome periferica obbligatorio")
	ErrPortInUse              = errors.New("porta I/O gia' assegnata")
	ErrPeripheralNotFound     = errors.New("periferica non collegata")
)

// PeripheralInput collega una callback a una porta input.
type PeripheralInput struct {
	Port    byte
	Handler InputCallback
}

// PeripheralOutput collega una callback a una porta output.
type PeripheralOutput struct {
	Port    byte
	Handler OutputCallback
}

// PeripheralBinding descrive tutte le porte possedute da una periferica.
type PeripheralBinding struct {
	Name    string
	Inputs  []PeripheralInput
	Outputs []PeripheralOutput
}

// PortBinding descrive una porta assegnata, utile per UI e diagnostica.
type PortBinding struct {
	Peripheral string
	Direction  IODirection
	Port       byte
}

// PeripheralBus gestisce ownership e conflitti sopra CallbackIO.
type PeripheralBus struct {
	io *CallbackIO
	mu sync.Mutex

	inputOwners  [256]string
	outputOwners [256]string
}

// NewPeripheralBus crea un gestore per un bus callback esistente.
func NewPeripheralBus(ioBus *CallbackIO) (*PeripheralBus, error) {
	if ioBus == nil {
		return nil, cpu.ErrNilIO
	}
	return &PeripheralBus{io: ioBus}, nil
}

// Attach valida l'intero binding prima di modificare il bus.
func (b *PeripheralBus) Attach(binding PeripheralBinding) error {
	if binding.Name == "" {
		return ErrPeripheralNameRequired
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	seenInputs := make(map[byte]struct{}, len(binding.Inputs))
	for _, input := range binding.Inputs {
		if err := cpu.ValidateInputPort(input.Port); err != nil {
			return err
		}
		if input.Handler == nil {
			return fmt.Errorf("periferica %q: callback input %d nil", binding.Name, input.Port)
		}
		if _, duplicate := seenInputs[input.Port]; duplicate {
			return fmt.Errorf("periferica %q: input %d duplicato", binding.Name, input.Port)
		}
		seenInputs[input.Port] = struct{}{}
		if owner := b.inputOwner(input.Port); owner != "" {
			return fmt.Errorf("%w: input %d posseduto da %s", ErrPortInUse, input.Port, owner)
		}
	}

	seenOutputs := make(map[byte]struct{}, len(binding.Outputs))
	for _, output := range binding.Outputs {
		if err := cpu.ValidateOutputPort(output.Port); err != nil {
			return err
		}
		if output.Handler == nil {
			return fmt.Errorf("periferica %q: callback output %d nil", binding.Name, output.Port)
		}
		if _, duplicate := seenOutputs[output.Port]; duplicate {
			return fmt.Errorf("periferica %q: output %d duplicato", binding.Name, output.Port)
		}
		seenOutputs[output.Port] = struct{}{}
		if owner := b.outputOwner(output.Port); owner != "" {
			return fmt.Errorf("%w: output %d posseduto da %s", ErrPortInUse, output.Port, owner)
		}
	}

	for _, input := range binding.Inputs {
		b.io.inputCallbacks[input.Port] = input.Handler
		b.inputOwners[input.Port] = binding.Name
	}
	for _, output := range binding.Outputs {
		b.io.outputCallbacks[output.Port] = output.Handler
		b.outputOwners[output.Port] = binding.Name
	}
	return nil
}

// Detach libera tutte le porte possedute dalla periferica.
func (b *PeripheralBus) Detach(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	found := false
	for port, owner := range b.inputOwners {
		if owner == name {
			b.inputOwners[port] = ""
			b.io.inputCallbacks[port] = nil
			found = true
		}
	}
	for index, owner := range b.outputOwners {
		if owner == name {
			b.outputOwners[index] = ""
			b.io.outputCallbacks[index] = nil
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrPeripheralNotFound, name)
	}
	return nil
}

// Bindings restituisce una fotografia ordinata delle assegnazioni.
func (b *PeripheralBus) Bindings() []PortBinding {
	b.mu.Lock()
	defer b.mu.Unlock()
	var bindings []PortBinding
	for port, owner := range b.inputOwners {
		if owner != "" {
			bindings = append(bindings, PortBinding{Peripheral: owner, Direction: IODirectionInput, Port: byte(port)})
		}
	}
	for index, owner := range b.outputOwners {
		if owner != "" {
			bindings = append(bindings, PortBinding{Peripheral: owner, Direction: IODirectionOutput, Port: byte(index)})
		}
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Direction != bindings[j].Direction {
			return bindings[i].Direction < bindings[j].Direction
		}
		return bindings[i].Port < bindings[j].Port
	})
	return bindings
}

func (b *PeripheralBus) inputOwner(port byte) string {
	if b.inputOwners[port] != "" {
		return b.inputOwners[port]
	}
	if b.io.inputCallbacks[port] != nil {
		return "callback-esterno"
	}
	return ""
}

func (b *PeripheralBus) outputOwner(port byte) string {
	if b.outputOwners[port] != "" {
		return b.outputOwners[port]
	}
	if b.io.outputCallbacks[port] != nil {
		return "callback-esterno"
	}
	return ""
}

// RegisterPeripheral e' un registro a 8 bit leggibile e scrivibile via I/O.
type RegisterPeripheral struct {
	mu    sync.Mutex
	value byte
}

func NewRegisterPeripheral(initial byte) *RegisterPeripheral {
	return &RegisterPeripheral{value: initial}
}

func (r *RegisterPeripheral) Value() byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.value
}

func (r *RegisterPeripheral) SetValue(value byte) {
	r.mu.Lock()
	r.value = value
	r.mu.Unlock()
}

// Attach collega input e output distinti allo stesso registro.
func (r *RegisterPeripheral) Attach(bus *PeripheralBus, name string, inputPort byte, outputPort byte) error {
	if bus == nil {
		return cpu.ErrNilIO
	}
	return bus.Attach(PeripheralBinding{
		Name: name,
		Inputs: []PeripheralInput{{
			Port: inputPort,
			Handler: func(_ byte, _ byte) byte {
				return r.Value()
			},
		}},
		Outputs: []PeripheralOutput{{
			Port: outputPort,
			Handler: func(_ byte, value byte) {
				r.SetValue(value)
			},
		}},
	})
}
