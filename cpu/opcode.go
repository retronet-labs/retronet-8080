package cpu

type ExecuteFunc func(c *CPU8080, mem Memory, io IO, inst Instruction) error

type MachineCycle string

const (
	CycleFetch       MachineCycle = "M1"
	CycleMemoryRead  MachineCycle = "MR"
	CycleMemoryWrite MachineCycle = "MW"
	CycleIORead      MachineCycle = "IOR"
	CycleIOWrite     MachineCycle = "IOW"
	CycleStackRead   MachineCycle = "SR"
	CycleStackWrite  MachineCycle = "SW"
)

type Opcode struct {
	Code       byte
	Mnemonic   string
	Length     byte
	MinStates  byte
	States     byte
	CycleCount byte
	Cycles     [5]MachineCycle
	Execute    ExecuteFunc
}

func (o Opcode) MachineCycles() []MachineCycle {
	cycles := make([]MachineCycle, o.CycleCount)
	copy(cycles, o.Cycles[:o.CycleCount])
	return cycles
}

type InstructionTiming struct {
	States      byte
	WaitStates  uint64
	CycleCount  byte
	Cycles      [5]MachineCycle
	Conditional bool
	Taken       bool
}

func (t InstructionTiming) MachineCycles() []MachineCycle {
	cycles := make([]MachineCycle, t.CycleCount)
	copy(cycles, t.Cycles[:t.CycleCount])
	return cycles
}

type Instruction struct {
	PC           uint16
	Opcode       Opcode
	Operands     [2]byte
	OperandCount byte
}

func (inst Instruction) wordOperand() uint16 {
	return uint16(inst.Operands[1])<<8 | uint16(inst.Operands[0])
}

func unimplementedExecute(_ *CPU8080, _ Memory, _ IO, inst Instruction) error {
	return &UnimplementedOpcodeError{
		PC:       inst.PC,
		Opcode:   inst.Opcode.Code,
		Mnemonic: inst.Opcode.Mnemonic,
		Length:   inst.Opcode.Length,
	}
}
