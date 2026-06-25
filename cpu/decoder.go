package cpu

import "fmt"

var decoder = buildDecoder()

func Decode(code byte) Opcode { return decoder[code] }

func OpcodeTable() [256]Opcode { return decoder }

func buildDecoder() [256]Opcode {
	var table [256]Opcode
	for i := range table {
		code := byte(i)
		execute := ExecuteFunc(execute8080)
		if isUnassignedOpcode(code) {
			execute = unimplementedExecute
		}
		minStates, states := stateRangeFor(code)
		cycles, cycleCount := cyclesFor(code)
		table[i] = Opcode{
			Code:       code,
			Mnemonic:   mnemonicFor(code),
			Length:     lengthFor(code),
			MinStates:  minStates,
			States:     states,
			CycleCount: cycleCount,
			Cycles:     cycles,
			Execute:    execute,
		}
	}
	return table
}

func isUnassignedOpcode(code byte) bool {
	switch code {
	case 0x08, 0x10, 0x18, 0x20, 0x28, 0x30, 0x38, 0xCB, 0xD9, 0xDD, 0xED, 0xFD:
		return true
	default:
		return false
	}
}

func lengthFor(code byte) byte {
	if isUnassignedOpcode(code) {
		return 1
	}
	switch {
	case code&0xC7 == 0x06:
		return 2
	case code&0xCF == 0x01:
		return 3
	case code == 0x22 || code == 0x2A || code == 0x32 || code == 0x3A:
		return 3
	case code == 0xC3 || code == 0xCD || code&0xC7 == 0xC2 || code&0xC7 == 0xC4:
		return 3
	case isImmediateALUOpcode(code) || code == 0xDB || code == 0xD3:
		return 2
	default:
		return 1
	}
}

func stateRangeFor(code byte) (byte, byte) {
	if isUnassignedOpcode(code) {
		return 0, 0
	}
	switch {
	case code == 0x00:
		return 4, 4
	case code == 0x76:
		return 7, 7
	case code&0xC0 == 0x40:
		if code&0x07 == byte(RegM) || (code>>3)&0x07 == byte(RegM) {
			return 7, 7
		}
		return 5, 5
	case code&0xC7 == 0x06:
		if (code>>3)&0x07 == byte(RegM) {
			return 10, 10
		}
		return 7, 7
	case code&0xC7 == 0x04 || code&0xC7 == 0x05:
		if (code>>3)&0x07 == byte(RegM) {
			return 10, 10
		}
		return 5, 5
	case code&0xCF == 0x01:
		return 10, 10
	case code == 0x02 || code == 0x12 || code == 0x0A || code == 0x1A:
		return 7, 7
	case code&0xCF == 0x03 || code&0xCF == 0x0B:
		return 5, 5
	case code&0xCF == 0x09:
		return 10, 10
	case code == 0x22 || code == 0x2A:
		return 16, 16
	case code == 0x32 || code == 0x3A:
		return 13, 13
	case code == 0xE3:
		return 18, 18
	case code == 0xE9 || code == 0xF9:
		return 5, 5
	case code == 0xEB:
		return 4, 4
	case code == 0xDB || code == 0xD3:
		return 10, 10
	case isImmediateALUOpcode(code):
		return 7, 7
	case code&0xC0 == 0x80:
		if code&0x07 == byte(RegM) {
			return 7, 7
		}
		return 4, 4
	case code&0xC7 == 0xC0:
		return 5, 11
	case code == 0xC9:
		return 10, 10
	case code&0xC7 == 0xC2 || code == 0xC3:
		return 10, 10
	case code&0xC7 == 0xC4:
		return 11, 17
	case code == 0xCD:
		return 17, 17
	case code&0xCF == 0xC1:
		return 10, 10
	case code&0xCF == 0xC5:
		return 11, 11
	case code&0xC7 == 0xC7:
		return 11, 11
	default:
		return 4, 4
	}
}

func cyclesFor(code byte) ([5]MachineCycle, byte) {
	cycles := [5]MachineCycle{CycleFetch}
	if isUnassignedOpcode(code) {
		return cycles, 1
	}
	switch {
	case code == 0x76:
		return cycles, 1
	case code&0xC0 == 0x40:
		dst := (code >> 3) & 0x07
		src := code & 0x07
		if dst == byte(RegM) {
			cycles[1] = CycleMemoryWrite
			return cycles, 2
		}
		if src == byte(RegM) {
			cycles[1] = CycleMemoryRead
			return cycles, 2
		}
		return cycles, 1
	case code == 0xD3:
		cycles[1], cycles[2] = CycleMemoryRead, CycleIOWrite
		return cycles, 3
	case code == 0xDB:
		cycles[1], cycles[2] = CycleMemoryRead, CycleIORead
		return cycles, 3
	case code&0xCF == 0xC5 || code == 0xCD || code&0xC7 == 0xC4 || code&0xC7 == 0xC7:
		cycles[1], cycles[2] = CycleStackWrite, CycleStackWrite
		if lengthFor(code) == 3 {
			cycles[3], cycles[4] = CycleMemoryRead, CycleMemoryRead
			return cycles, 5
		}
		return cycles, 3
	case code&0xCF == 0xC1 || code == 0xC9 || code&0xC7 == 0xC0:
		cycles[1], cycles[2] = CycleStackRead, CycleStackRead
		return cycles, 3
	case code == 0x32 || code == 0x22:
		cycles[1], cycles[2], cycles[3] = CycleMemoryRead, CycleMemoryRead, CycleMemoryWrite
		return cycles, 4
	case code == 0x3A || code == 0x2A:
		cycles[1], cycles[2], cycles[3] = CycleMemoryRead, CycleMemoryRead, CycleMemoryRead
		if code == 0x2A {
			cycles[4] = CycleMemoryRead
			return cycles, 5
		}
		return cycles, 4
	case lengthFor(code) == 3:
		cycles[1], cycles[2] = CycleMemoryRead, CycleMemoryRead
		return cycles, 3
	case lengthFor(code) == 2:
		cycles[1] = CycleMemoryRead
		return cycles, 2
	default:
		return cycles, 1
	}
}

func isImmediateALUOpcode(code byte) bool {
	switch code {
	case 0xC6, 0xCE, 0xD6, 0xDE, 0xE6, 0xEE, 0xF6, 0xFE:
		return true
	default:
		return false
	}
}

func mnemonicFor(code byte) string {
	if isUnassignedOpcode(code) {
		return fmt.Sprintf("??? 0x%02X", code)
	}
	switch {
	case code == 0x00:
		return "NOP"
	case code == 0x76:
		return "HLT"
	case code&0xC0 == 0x40:
		return fmt.Sprintf("MOV %s,%s", registerName((code>>3)&0x07), registerName(code&0x07))
	case code&0xC7 == 0x06:
		return fmt.Sprintf("MVI %s", registerName((code>>3)&0x07))
	case code&0xCF == 0x01:
		return fmt.Sprintf("LXI %s", pairName((code>>4)&0x03, false))
	case code&0xC7 == 0x04:
		return fmt.Sprintf("INR %s", registerName((code>>3)&0x07))
	case code&0xC7 == 0x05:
		return fmt.Sprintf("DCR %s", registerName((code>>3)&0x07))
	case code&0xCF == 0x03:
		return fmt.Sprintf("INX %s", pairName((code>>4)&0x03, false))
	case code&0xCF == 0x0B:
		return fmt.Sprintf("DCX %s", pairName((code>>4)&0x03, false))
	case code&0xCF == 0x09:
		return fmt.Sprintf("DAD %s", pairName((code>>4)&0x03, false))
	case code&0xC0 == 0x80:
		return fmt.Sprintf("%s %s", aluName((code>>3)&0x07), registerName(code&0x07))
	case isImmediateALUOpcode(code):
		return immediateALUName((code >> 3) & 0x07)
	case code&0xC7 == 0xC2:
		return "J" + conditionName((code>>3)&0x07)
	case code&0xC7 == 0xC4:
		return "C" + conditionName((code>>3)&0x07)
	case code&0xC7 == 0xC0:
		return "R" + conditionName((code>>3)&0x07)
	case code&0xC7 == 0xC7:
		return fmt.Sprintf("RST %d", (code>>3)&0x07)
	case code&0xCF == 0xC1:
		return "POP " + pairName((code>>4)&0x03, true)
	case code&0xCF == 0xC5:
		return "PUSH " + pairName((code>>4)&0x03, true)
	default:
		return fixedMnemonic(code)
	}
}

func fixedMnemonic(code byte) string {
	switch code {
	case 0x02:
		return "STAX B"
	case 0x0A:
		return "LDAX B"
	case 0x12:
		return "STAX D"
	case 0x1A:
		return "LDAX D"
	case 0x07:
		return "RLC"
	case 0x0F:
		return "RRC"
	case 0x17:
		return "RAL"
	case 0x1F:
		return "RAR"
	case 0x22:
		return "SHLD"
	case 0x2A:
		return "LHLD"
	case 0x27:
		return "DAA"
	case 0x2F:
		return "CMA"
	case 0x32:
		return "STA"
	case 0x3A:
		return "LDA"
	case 0x37:
		return "STC"
	case 0x3F:
		return "CMC"
	case 0xC3:
		return "JMP"
	case 0xC9:
		return "RET"
	case 0xCD:
		return "CALL"
	case 0xD3:
		return "OUT"
	case 0xDB:
		return "IN"
	case 0xE3:
		return "XTHL"
	case 0xE9:
		return "PCHL"
	case 0xEB:
		return "XCHG"
	case 0xF3:
		return "DI"
	case 0xF9:
		return "SPHL"
	case 0xFB:
		return "EI"
	default:
		return fmt.Sprintf("NOP 0x%02X", code)
	}
}

func registerName(code byte) string {
	return [...]string{"B", "C", "D", "E", "H", "L", "M", "A"}[code&0x07]
}

func pairName(code byte, psw bool) string {
	if psw && code&0x03 == 3 {
		return "PSW"
	}
	return [...]string{"B", "D", "H", "SP"}[code&0x03]
}

func conditionName(code byte) string {
	return [...]string{"NZ", "Z", "NC", "C", "PO", "PE", "P", "M"}[code&0x07]
}

func aluName(group byte) string {
	return [...]string{"ADD", "ADC", "SUB", "SBB", "ANA", "XRA", "ORA", "CMP"}[group&0x07]
}

func immediateALUName(group byte) string {
	return [...]string{"ADI", "ACI", "SUI", "SBI", "ANI", "XRI", "ORI", "CPI"}[group&0x07]
}
