package cpu

import "fmt"

func execute8080(c *CPU8080, mem Memory, io IO, inst Instruction) error {
	code := inst.Opcode.Code
	switch {
	case code == 0x00:
		return nil
	case code == 0x76:
		c.Halted = true
		c.Stopped = true
		return nil
	case code&0xC0 == 0x40:
		return executeMOV(c, mem, code)
	case code&0xC7 == 0x06:
		return c.writeRegister(Register((code>>3)&0x07), inst.Operands[0], mem)
	case code&0xCF == 0x01:
		c.setPair(RegisterPair((code>>4)&0x03), inst.wordOperand())
		return nil
	case code&0xC7 == 0x04:
		r := Register((code >> 3) & 0x07)
		value, err := c.readRegister(r, mem)
		if err != nil {
			return err
		}
		return c.writeRegister(r, c.increment(value), mem)
	case code&0xC7 == 0x05:
		r := Register((code >> 3) & 0x07)
		value, err := c.readRegister(r, mem)
		if err != nil {
			return err
		}
		return c.writeRegister(r, c.decrement(value), mem)
	case code&0xCF == 0x03:
		pair := RegisterPair((code >> 4) & 0x03)
		c.setPair(pair, c.pair(pair)+1)
		return nil
	case code&0xCF == 0x0B:
		pair := RegisterPair((code >> 4) & 0x03)
		c.setPair(pair, c.pair(pair)-1)
		return nil
	case code&0xCF == 0x09:
		c.dad(c.pair(RegisterPair((code >> 4) & 0x03)))
		return nil
	case code&0xC0 == 0x80:
		value, err := c.readRegister(Register(code&0x07), mem)
		if err != nil {
			return err
		}
		c.executeALU((code>>3)&0x07, value)
		return nil
	case isImmediateALUOpcode(code):
		c.executeALU((code>>3)&0x07, inst.Operands[0])
		return nil
	case code&0xC7 == 0xC2:
		if c.conditionTaken(Condition((code >> 3) & 0x07)) {
			c.setPC(inst.wordOperand())
		}
		return nil
	case code&0xC7 == 0xC4:
		if c.conditionTaken(Condition((code >> 3) & 0x07)) {
			c.call(mem, inst.wordOperand())
		}
		return nil
	case code&0xC7 == 0xC0:
		if c.conditionTaken(Condition((code >> 3) & 0x07)) {
			c.setPC(c.popWord(mem))
		}
		return nil
	case code&0xC7 == 0xC7:
		c.call(mem, uint16((code>>3)&0x07)<<3)
		return nil
	case code&0xCF == 0xC1:
		c.pop(RegisterPair((code>>4)&0x03), mem)
		return nil
	case code&0xCF == 0xC5:
		c.push(RegisterPair((code>>4)&0x03), mem)
		return nil
	}

	switch code {
	case 0x02:
		mem.Write(c.BC(), c.A)
	case 0x12:
		mem.Write(c.DE(), c.A)
	case 0x0A:
		c.A = mem.Read(c.BC())
	case 0x1A:
		c.A = mem.Read(c.DE())
	case 0x07:
		c.Carry = c.A&0x80 != 0
		c.A = c.A<<1 | boolByte(c.Carry)
	case 0x0F:
		c.Carry = c.A&0x01 != 0
		c.A = c.A>>1 | boolByte(c.Carry)<<7
	case 0x17:
		oldCarry := c.Carry
		c.Carry = c.A&0x80 != 0
		c.A = c.A<<1 | boolByte(oldCarry)
	case 0x1F:
		oldCarry := c.Carry
		c.Carry = c.A&0x01 != 0
		c.A = c.A>>1 | boolByte(oldCarry)<<7
	case 0x22:
		addr := inst.wordOperand()
		mem.Write(addr, c.L)
		mem.Write(addr+1, c.H)
	case 0x2A:
		addr := inst.wordOperand()
		c.L = mem.Read(addr)
		c.H = mem.Read(addr + 1)
	case 0x27:
		c.daa()
	case 0x2F:
		c.A = ^c.A
	case 0x32:
		mem.Write(inst.wordOperand(), c.A)
	case 0x3A:
		c.A = mem.Read(inst.wordOperand())
	case 0x37:
		c.Carry = true
	case 0x3F:
		c.Carry = !c.Carry
	case 0xC3:
		c.setPC(inst.wordOperand())
	case 0xC9:
		c.setPC(c.popWord(mem))
	case 0xCD:
		c.call(mem, inst.wordOperand())
	case 0xD3:
		if io == nil {
			return ErrNilIO
		}
		io.Output(inst.Operands[0], c.A)
	case 0xDB:
		if io == nil {
			return ErrNilIO
		}
		c.A = io.Input(inst.Operands[0])
	case 0xE3:
		low := mem.Read(c.SP)
		high := mem.Read(c.SP + 1)
		mem.Write(c.SP, c.L)
		mem.Write(c.SP+1, c.H)
		c.L, c.H = low, high
	case 0xE9:
		c.setPC(c.HL())
	case 0xEB:
		c.D, c.E, c.H, c.L = c.H, c.L, c.D, c.E
	case 0xF3:
		c.InterruptsEnabled = false
	case 0xF9:
		c.SP = c.HL()
	case 0xFB:
		c.InterruptsEnabled = true
	default:
		return fmt.Errorf("%w: 0x%02X", ErrUnimplementedOpcode, code)
	}
	return nil
}

func executeMOV(c *CPU8080, mem Memory, code byte) error {
	value, err := c.readRegister(Register(code&0x07), mem)
	if err != nil {
		return err
	}
	return c.writeRegister(Register((code>>3)&0x07), value, mem)
}

func (c *CPU8080) call(mem Memory, target uint16) {
	c.pushWord(mem, c.PC)
	c.setPC(target)
}

func (c *CPU8080) push(pair RegisterPair, mem Memory) {
	switch pair {
	case PairBC:
		c.pushWord(mem, c.BC())
	case PairDE:
		c.pushWord(mem, c.DE())
	case PairHL:
		c.pushWord(mem, c.HL())
	default:
		c.pushWord(mem, uint16(c.A)<<8|uint16(c.FlagsByte()))
	}
}

func (c *CPU8080) pop(pair RegisterPair, mem Memory) {
	value := c.popWord(mem)
	switch pair {
	case PairBC:
		c.SetBC(value)
	case PairDE:
		c.SetDE(value)
	case PairHL:
		c.SetHL(value)
	default:
		c.A = byte(value >> 8)
		c.SetFlagsByte(byte(value))
	}
}

func (c *CPU8080) pushWord(mem Memory, value uint16) {
	c.SP--
	mem.Write(c.SP, byte(value>>8))
	c.SP--
	mem.Write(c.SP, byte(value))
}

func (c *CPU8080) popWord(mem Memory) uint16 {
	low := mem.Read(c.SP)
	c.SP++
	high := mem.Read(c.SP)
	c.SP++
	return uint16(high)<<8 | uint16(low)
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}
