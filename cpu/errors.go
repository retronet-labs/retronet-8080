package cpu

import (
	"errors"
	"fmt"
)

var (
	ErrNilMemory             = errors.New("memoria 8080 non inizializzata")
	ErrNilIO                 = errors.New("bus I/O 8080 non inizializzato")
	ErrCPUStopped            = errors.New("cpu 8080 ferma")
	ErrUnimplementedOpcode   = errors.New("opcode 8080 non implementato")
	ErrInvalidJamInstruction = errors.New("jam instruction 8080 non valida")
)

// UnimplementedOpcodeError descrive un byte non assegnato dall'ISA 8080.
type UnimplementedOpcodeError struct {
	PC       uint16
	Opcode   byte
	Mnemonic string
	Length   byte
}

func (e *UnimplementedOpcodeError) Error() string {
	return fmt.Sprintf("%v: pc=0x%04X opcode=0x%02X mnemonic=%s length=%d",
		ErrUnimplementedOpcode, e.PC, e.Opcode, e.Mnemonic, e.Length)
}

func (e *UnimplementedOpcodeError) Unwrap() error {
	return ErrUnimplementedOpcode
}
