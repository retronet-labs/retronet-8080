package cpu

import "fmt"

type Disassembly struct {
	PC      uint16
	Opcode  Opcode
	Bytes   [3]byte
	Length  byte
	Operand uint16
	NextPC  uint16
}

func Disassemble(mem Memory, pc uint16) (Disassembly, error) {
	if mem == nil {
		return Disassembly{}, ErrNilMemory
	}
	code := mem.Read(pc)
	op := Decode(code)
	d := Disassembly{PC: pc, Opcode: op, Length: op.Length, NextPC: pc + uint16(op.Length)}
	if d.Length == 0 {
		d.Length = 1
		d.NextPC = pc + 1
	}
	for i := byte(0); i < d.Length; i++ {
		d.Bytes[i] = mem.Read(pc + uint16(i))
	}
	if d.Length == 2 {
		d.Operand = uint16(d.Bytes[1])
	} else if d.Length == 3 {
		d.Operand = uint16(d.Bytes[2])<<8 | uint16(d.Bytes[1])
	}
	return d, nil
}

func (d Disassembly) String() string {
	bytes := ""
	for i := byte(0); i < d.Length; i++ {
		if i > 0 {
			bytes += " "
		}
		bytes += fmt.Sprintf("%02X", d.Bytes[i])
	}
	return fmt.Sprintf("%04X: %-8s %s", d.PC, bytes, d.text())
}

func (d Disassembly) text() string {
	code := d.Opcode.Code
	if isUnassignedOpcode(code) {
		return fmt.Sprintf("??? 0x%02X", code)
	}
	switch {
	case code&0xC7 == 0x06:
		return fmt.Sprintf("MVI %s,#0x%02X", registerName((code>>3)&0x07), byte(d.Operand))
	case code&0xCF == 0x01:
		return fmt.Sprintf("LXI %s,#0x%04X", pairName((code>>4)&0x03, false), d.Operand)
	case code == 0x22:
		return fmt.Sprintf("SHLD 0x%04X", d.Operand)
	case code == 0x2A:
		return fmt.Sprintf("LHLD 0x%04X", d.Operand)
	case code == 0x32:
		return fmt.Sprintf("STA 0x%04X", d.Operand)
	case code == 0x3A:
		return fmt.Sprintf("LDA 0x%04X", d.Operand)
	case code == 0xC3:
		return fmt.Sprintf("JMP 0x%04X", d.Operand)
	case code == 0xCD:
		return fmt.Sprintf("CALL 0x%04X", d.Operand)
	case code&0xC7 == 0xC2:
		return fmt.Sprintf("J%s 0x%04X", conditionName((code>>3)&0x07), d.Operand)
	case code&0xC7 == 0xC4:
		return fmt.Sprintf("C%s 0x%04X", conditionName((code>>3)&0x07), d.Operand)
	case isImmediateALUOpcode(code):
		return fmt.Sprintf("%s #0x%02X", immediateALUName((code>>3)&0x07), byte(d.Operand))
	case code == 0xDB:
		return fmt.Sprintf("IN 0x%02X", byte(d.Operand))
	case code == 0xD3:
		return fmt.Sprintf("OUT 0x%02X", byte(d.Operand))
	default:
		return mnemonicFor(code)
	}
}
