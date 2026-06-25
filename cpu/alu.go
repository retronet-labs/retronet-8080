package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8080"

func (c *CPU8080) executeALU(group byte, value byte) {
	result, flags := i8080.ALU(group, c.A, value, c.Carry)
	c.applyALUFlags(flags)
	if group&0x07 != i8080.GroupCMP {
		c.A = result
	}
}

func (c *CPU8080) applyALUFlags(flags i8080.Flags) {
	c.Carry = flags.Carry
	c.Zero = flags.Zero
	c.Sign = flags.Sign
	c.Parity = flags.Parity
	c.AuxiliaryCarry = flags.AuxiliaryCarry
}

func (c *CPU8080) applyNonCarryFlags(flags i8080.Flags) {
	c.Zero = flags.Zero
	c.Sign = flags.Sign
	c.Parity = flags.Parity
	c.AuxiliaryCarry = flags.AuxiliaryCarry
}

func (c *CPU8080) increment(value byte) byte {
	result, flags := i8080.Increment(value)
	c.applyNonCarryFlags(flags)
	return result
}

func (c *CPU8080) decrement(value byte) byte {
	result, flags := i8080.Decrement(value)
	c.applyNonCarryFlags(flags)
	return result
}

func (c *CPU8080) dad(value uint16) {
	result, carry := i8080.Add16(c.HL(), value)
	c.SetHL(result)
	c.Carry = carry
}

func (c *CPU8080) daa() {
	oldA := c.A
	correction := byte(0)
	carry := c.Carry
	if oldA&0x0F > 9 || c.AuxiliaryCarry {
		correction |= 0x06
	}
	if oldA > 0x99 || c.Carry {
		correction |= 0x60
		carry = true
	}
	result, flags := i8080.ALU(i8080.GroupADD, oldA, correction, false)
	c.A = result
	c.Zero = flags.Zero
	c.Sign = flags.Sign
	c.Parity = flags.Parity
	c.AuxiliaryCarry = flags.AuxiliaryCarry
	c.Carry = carry || flags.Carry
}
