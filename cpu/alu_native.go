package cpu

import (
	"math/bits"

	"github.com/retronet-labs/retronet-hardware/bridge/i8080"
)

// nativeBackend riproduce la semantica aritmetico-logica dell'8080 con i normali
// operatori di Go. È funzionalmente identico al backend a porte (Gate), ma più
// rapido (8080EXM ≈1,8×): utile per le diagnostiche esaustive e come oracolo del
// differenziale.
type nativeBackend struct{}

func nativeParityEven(v byte) bool { return bits.OnesCount8(v)%2 == 0 }

// carryInto indica se l'addizione a+b+cy genera un riporto entrante nel bit dato.
func carryInto(bitNo, a, b, cy int) bool {
	res := a + b + cy
	return (res^a^b)&(1<<bitNo) != 0
}

func nativeAdd(a, val byte, cy bool) (byte, i8080.Flags) {
	c := 0
	if cy {
		c = 1
	}
	out := byte(int(a) + int(val) + c)
	return out, i8080.Flags{
		Carry:          carryInto(8, int(a), int(val), c),
		AuxiliaryCarry: carryInto(4, int(a), int(val), c),
		Zero:           out == 0,
		Sign:           out&0x80 != 0,
		Parity:         nativeParityEven(out),
	}
}

// nativeSub calcola SUB/SBB/CMP come ADD(a, ^val, !borrow): il Carry finale è il
// prestito (riporto invertito), mentre l'Auxiliary Carry resta il riporto del
// bit 3 senza inversione.
func nativeSub(a, val byte, borrow bool) (byte, i8080.Flags) {
	nc := 1
	if borrow {
		nc = 0
	}
	nval := int(^val)
	out := byte(int(a) + nval + nc)
	return out, i8080.Flags{
		Carry:          !carryInto(8, int(a), nval, nc),
		AuxiliaryCarry: carryInto(4, int(a), nval, nc),
		Zero:           out == 0,
		Sign:           out&0x80 != 0,
		Parity:         nativeParityEven(out),
	}
}

func nativeLogic(a, val, group byte) (byte, i8080.Flags) {
	var out byte
	var aux bool
	switch group & 0x07 {
	case i8080.GroupANA:
		out = a & val
		aux = (a|val)&0x08 != 0 // quirk 8080: AC = bit 3 di (A OR value)
	case i8080.GroupXRA:
		out = a ^ val
	default: // GroupORA
		out = a | val
	}
	return out, i8080.Flags{
		Zero:           out == 0,
		Sign:           out&0x80 != 0,
		Parity:         nativeParityEven(out),
		AuxiliaryCarry: aux,
	}
}

func (nativeBackend) ALU(group, a, value byte, carryIn bool) (byte, i8080.Flags) {
	switch group & 0x07 {
	case i8080.GroupADD:
		return nativeAdd(a, value, false)
	case i8080.GroupADC:
		return nativeAdd(a, value, carryIn)
	case i8080.GroupSUB:
		return nativeSub(a, value, false)
	case i8080.GroupSBB:
		return nativeSub(a, value, carryIn)
	case i8080.GroupANA, i8080.GroupXRA, i8080.GroupORA:
		return nativeLogic(a, value, group)
	default: // GroupCMP: come SUB, il risultato lo scarta il chiamante
		return nativeSub(a, value, false)
	}
}

func (n nativeBackend) Increment(value byte) (byte, i8080.Flags) {
	result, flags := n.ALU(i8080.GroupADD, value, 1, false)
	flags.Carry = false // INR non tocca il Carry
	return result, flags
}

func (n nativeBackend) Decrement(value byte) (byte, i8080.Flags) {
	result, flags := n.ALU(i8080.GroupSUB, value, 1, false)
	flags.Carry = false // DCR non tocca il Carry
	return result, flags
}

func (nativeBackend) Add16(a, value uint16) (uint16, bool) {
	sum := uint32(a) + uint32(value)
	return uint16(sum), sum>>16 != 0
}
