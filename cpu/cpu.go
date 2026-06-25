package cpu

// CPU8080 rappresenta lo stato visibile e i contatori didattici dell'Intel 8080.
type CPU8080 struct {
	A byte
	B byte
	C byte
	D byte
	E byte
	H byte
	L byte

	Carry          bool
	Zero           bool
	Sign           bool
	Parity         bool
	AuxiliaryCarry bool

	PC uint16
	SP uint16

	Halted            bool
	Stopped           bool
	InterruptsEnabled bool

	InstructionCount  uint64
	StateCount        uint64
	WaitStateCount    uint64
	LastTiming        InstructionTiming
	pendingWaitStates uint64
}

// NewCPU8080 crea una CPU nello stato di reset deterministico usato dal progetto.
func NewCPU8080() *CPU8080 {
	c := &CPU8080{}
	c.Reset()
	return c
}

// Reset azzera registri, flag e contatori. A differenza dell'8008, l'8080 parte
// eseguibile: non entra in stopped storico e non richiede jam instruction.
func (c *CPU8080) Reset() {
	*c = CPU8080{}
}

func (c *CPU8080) setPC(addr uint16) { c.PC = addr }

func (c *CPU8080) HL() uint16 { return uint16(c.H)<<8 | uint16(c.L) }

func (c *CPU8080) BC() uint16 { return uint16(c.B)<<8 | uint16(c.C) }

func (c *CPU8080) DE() uint16 { return uint16(c.D)<<8 | uint16(c.E) }

func (c *CPU8080) SetHL(value uint16) {
	c.H = byte(value >> 8)
	c.L = byte(value)
}

func (c *CPU8080) SetBC(value uint16) {
	c.B = byte(value >> 8)
	c.C = byte(value)
}

func (c *CPU8080) SetDE(value uint16) {
	c.D = byte(value >> 8)
	c.E = byte(value)
}

func (c *CPU8080) pair(pair RegisterPair) uint16 {
	switch pair {
	case PairBC:
		return c.BC()
	case PairDE:
		return c.DE()
	case PairHL:
		return c.HL()
	default:
		return c.SP
	}
}

func (c *CPU8080) setPair(pair RegisterPair, value uint16) {
	switch pair {
	case PairBC:
		c.SetBC(value)
	case PairDE:
		c.SetDE(value)
	case PairHL:
		c.SetHL(value)
	default:
		c.SP = value
	}
}

func (c *CPU8080) readRegister(r Register, mem Memory) (byte, error) {
	switch r & 0x07 {
	case RegB:
		return c.B, nil
	case RegC:
		return c.C, nil
	case RegD:
		return c.D, nil
	case RegE:
		return c.E, nil
	case RegH:
		return c.H, nil
	case RegL:
		return c.L, nil
	case RegM:
		if mem == nil {
			return 0, ErrNilMemory
		}
		return mem.Read(c.HL()), nil
	default:
		return c.A, nil
	}
}

func (c *CPU8080) writeRegister(r Register, value byte, mem Memory) error {
	switch r & 0x07 {
	case RegB:
		c.B = value
	case RegC:
		c.C = value
	case RegD:
		c.D = value
	case RegE:
		c.E = value
	case RegH:
		c.H = value
	case RegL:
		c.L = value
	case RegM:
		if mem == nil {
			return ErrNilMemory
		}
		mem.Write(c.HL(), value)
	default:
		c.A = value
	}
	return nil
}

// FlagsByte produce il byte PSW salvato da PUSH PSW.
func (c *CPU8080) FlagsByte() byte {
	var f byte = 0x02
	if c.Sign {
		f |= 0x80
	}
	if c.Zero {
		f |= 0x40
	}
	if c.AuxiliaryCarry {
		f |= 0x10
	}
	if c.Parity {
		f |= 0x04
	}
	if c.Carry {
		f |= 0x01
	}
	return f
}

// SetFlagsByte ripristina i flag dal byte PSW caricato da POP PSW.
func (c *CPU8080) SetFlagsByte(f byte) {
	c.Sign = f&0x80 != 0
	c.Zero = f&0x40 != 0
	c.AuxiliaryCarry = f&0x10 != 0
	c.Parity = f&0x04 != 0
	c.Carry = f&0x01 != 0
}

func (c *CPU8080) conditionTaken(cond Condition) bool {
	switch cond & 0x07 {
	case CondNZ:
		return !c.Zero
	case CondZ:
		return c.Zero
	case CondNC:
		return !c.Carry
	case CondC:
		return c.Carry
	case CondPO:
		return !c.Parity
	case CondPE:
		return c.Parity
	case CondP:
		return !c.Sign
	default:
		return c.Sign
	}
}
