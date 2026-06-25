package cpu

// Step esegue fetch-decode-execute di una singola istruzione 8080.
func (c *CPU8080) Step(mem Memory, io IO) error {
	if c.Halted || c.Stopped {
		return ErrCPUStopped
	}
	if mem == nil {
		return ErrNilMemory
	}

	pcBefore := c.PC
	code := c.fetch(mem)
	op := Decode(code)
	inst := Instruction{PC: pcBefore, Opcode: op}
	for i := byte(1); i < op.Length; i++ {
		inst.Operands[i-1] = c.fetch(mem)
		inst.OperandCount++
	}

	timing := c.instructionTiming(op)
	if err := op.Execute(c, mem, io, inst); err != nil {
		return err
	}
	c.recordTiming(timing)
	return nil
}

func (c *CPU8080) fetch(mem Memory) byte {
	value := mem.Read(c.PC)
	c.PC++
	return value
}

func (c *CPU8080) instructionTiming(op Opcode) InstructionTiming {
	timing := InstructionTiming{
		States:     op.States,
		CycleCount: op.CycleCount,
		Cycles:     op.Cycles,
		Taken:      true,
	}
	code := op.Code
	if code&0xC7 == 0xC0 || code&0xC7 == 0xC4 {
		timing.Conditional = true
		timing.Taken = c.conditionTaken(Condition((code >> 3) & 0x07))
		if !timing.Taken {
			timing.States = op.MinStates
		}
	}
	return timing
}

func (c *CPU8080) recordTiming(timing InstructionTiming) {
	timing.WaitStates = c.pendingWaitStates
	c.pendingWaitStates = 0
	c.InstructionCount++
	c.StateCount += uint64(timing.States)
	c.LastTiming = timing
}

func (c *CPU8080) RecordWaitState() {
	c.StateCount++
	c.WaitStateCount++
	c.pendingWaitStates++
}

// Jam esegue una istruzione fornita dall'esterno senza fetch da memoria.
func (c *CPU8080) Jam(mem Memory, io IO, code byte, operands ...byte) error {
	if mem == nil {
		return ErrNilMemory
	}
	op := Decode(code)
	want := int(op.Length - 1)
	if len(operands) != want {
		return ErrInvalidJamInstruction
	}
	inst := Instruction{PC: c.PC, Opcode: op, OperandCount: byte(want)}
	copy(inst.Operands[:], operands)
	c.Halted = false
	c.Stopped = false
	timing := c.instructionTiming(op)
	if err := op.Execute(c, mem, io, inst); err != nil {
		return err
	}
	c.recordTiming(timing)
	return nil
}
