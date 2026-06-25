package cpu

func regBits(r Register) byte { return byte(r) & 0x07 }

func pairBits(p RegisterPair) byte { return byte(p) & 0x03 }

func condBits(c Condition) byte { return byte(c) & 0x07 }

func NOP() byte { return 0x00 }

func HLT() byte { return 0x76 }

func MOV(dst, src Register) byte { return 0x40 | regBits(dst)<<3 | regBits(src) }

func MVI(dst Register) byte { return 0x06 | regBits(dst)<<3 }

func LXI(pair RegisterPair) byte { return 0x01 | pairBits(pair)<<4 }

func LDA() byte { return 0x3A }

func STA() byte { return 0x32 }

func LHLD() byte { return 0x2A }

func SHLD() byte { return 0x22 }

func LDAX(pair RegisterPair) byte {
	if pair == PairDE {
		return 0x1A
	}
	return 0x0A
}

func STAX(pair RegisterPair) byte {
	if pair == PairDE {
		return 0x12
	}
	return 0x02
}

func XCHG() byte { return 0xEB }

func XTHL() byte { return 0xE3 }

func SPHL() byte { return 0xF9 }

func PCHL() byte { return 0xE9 }

func INR(r Register) byte { return 0x04 | regBits(r)<<3 }

func DCR(r Register) byte { return 0x05 | regBits(r)<<3 }

func INX(pair RegisterPair) byte { return 0x03 | pairBits(pair)<<4 }

func DCX(pair RegisterPair) byte { return 0x0B | pairBits(pair)<<4 }

func DAD(pair RegisterPair) byte { return 0x09 | pairBits(pair)<<4 }

func ADD(src Register) byte { return 0x80 | regBits(src) }

func ADC(src Register) byte { return 0x88 | regBits(src) }

func SUB(src Register) byte { return 0x90 | regBits(src) }

func SBB(src Register) byte { return 0x98 | regBits(src) }

func ANA(src Register) byte { return 0xA0 | regBits(src) }

func XRA(src Register) byte { return 0xA8 | regBits(src) }

func ORA(src Register) byte { return 0xB0 | regBits(src) }

func CMP(src Register) byte { return 0xB8 | regBits(src) }

func ADI() byte { return 0xC6 }

func ACI() byte { return 0xCE }

func SUI() byte { return 0xD6 }

func SBI() byte { return 0xDE }

func ANI() byte { return 0xE6 }

func XRI() byte { return 0xEE }

func ORI() byte { return 0xF6 }

func CPI() byte { return 0xFE }

func DAA() byte { return 0x27 }

func CMA() byte { return 0x2F }

func STC() byte { return 0x37 }

func CMC() byte { return 0x3F }

func RLC() byte { return 0x07 }

func RRC() byte { return 0x0F }

func RAL() byte { return 0x17 }

func RAR() byte { return 0x1F }

func JMP() byte { return 0xC3 }

func J(cond Condition) byte { return 0xC2 | condBits(cond)<<3 }

func CALL() byte { return 0xCD }

func C(cond Condition) byte { return 0xC4 | condBits(cond)<<3 }

func RET() byte { return 0xC9 }

func R(cond Condition) byte { return 0xC0 | condBits(cond)<<3 }

func RST(n byte) byte { return 0xC7 | ((n & 0x07) << 3) }

func PUSH(pair RegisterPair) byte { return 0xC5 | pairBits(pair)<<4 }

func POP(pair RegisterPair) byte { return 0xC1 | pairBits(pair)<<4 }

func IN() byte { return 0xDB }

func OUT() byte { return 0xD3 }

func EI() byte { return 0xFB }

func DI() byte { return 0xF3 }
