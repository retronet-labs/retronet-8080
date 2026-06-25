package cpu

import (
	"errors"
	"testing"
)

func TestResetStartsRunnable(t *testing.T) {
	c := NewCPU8080()
	if c.Halted || c.Stopped || c.InterruptsEnabled || c.PC != 0 || c.SP != 0 {
		t.Fatalf("reset state = %+v", c)
	}
}

func TestMemoryAndPCWrapAt16Bit(t *testing.T) {
	c := NewCPU8080()
	mem := NewFlatMemory()
	mem.Data[0xFFFF] = NOP()
	c.PC = 0xFFFF
	if err := c.Step(mem, nil); err != nil {
		t.Fatal(err)
	}
	if c.PC != 0 {
		t.Fatalf("PC=0x%04X, want wrap to 0", c.PC)
	}
}

func TestLoadMoveALUAndFlags(t *testing.T) {
	c := NewCPU8080()
	mem := NewFlatMemory()
	program := []byte{MVI(RegA), 0xFF, ADI(), 0x01, HLT()}
	copy(mem.Data[:], program)

	runUntilStopped(t, c, mem, NewPorts(), 8)

	if c.A != 0 || !c.Carry || !c.Zero || c.Sign || !c.Parity || !c.AuxiliaryCarry {
		t.Fatalf("A=0x%02X C=%v Z=%v S=%v P=%v AC=%v", c.A, c.Carry, c.Zero, c.Sign, c.Parity, c.AuxiliaryCarry)
	}
}

func TestINRDCRPreserveCarry(t *testing.T) {
	c := NewCPU8080()
	c.Carry = true
	mem := NewFlatMemory()
	program := []byte{MVI(RegB), 0xFF, INR(RegB), DCR(RegB), HLT()}
	copy(mem.Data[:], program)

	runUntilStopped(t, c, mem, nil, 8)

	if c.B != 0xFF || !c.Carry || !c.Sign || c.Zero {
		t.Fatalf("B=0x%02X C=%v Z=%v S=%v", c.B, c.Carry, c.Zero, c.Sign)
	}
}

func TestStackCallReturnUsesMemory(t *testing.T) {
	c := NewCPU8080()
	mem := NewFlatMemory()
	program := []byte{
		CALL(), 0x06, 0x00,
		HLT(), 0x00, 0x00,
		MVI(RegA), 0x42,
		RET(),
	}
	copy(mem.Data[:], program)

	runUntilStopped(t, c, mem, nil, 16)

	if c.A != 0x42 || c.SP != 0 || c.PC != 0x0004 {
		t.Fatalf("A=0x%02X SP=0x%04X PC=0x%04X", c.A, c.SP, c.PC)
	}
	if mem.Read(0xFFFE) != 0x03 || mem.Read(0xFFFF) != 0x00 {
		t.Fatalf("return bytes low=0x%02X high=0x%02X", mem.Read(0xFFFE), mem.Read(0xFFFF))
	}
}

func TestIOAndInterruptEnable(t *testing.T) {
	c := NewCPU8080()
	mem := NewFlatMemory()
	ports := NewPorts()
	_ = ports.SetInput(0x42, 0xA5)
	program := []byte{EI(), IN(), 0x42, OUT(), 0x24, DI(), HLT()}
	copy(mem.Data[:], program)

	runUntilStopped(t, c, mem, ports, 16)

	if c.A != 0xA5 || ports.OutputPorts[0x24] != 0xA5 || c.InterruptsEnabled {
		t.Fatalf("A=0x%02X OUT=0x%02X IE=%v", c.A, ports.OutputPorts[0x24], c.InterruptsEnabled)
	}
}

func TestUnassignedOpcodeReturnsTypedError(t *testing.T) {
	c := NewCPU8080()
	mem := NewFlatMemory()
	mem.Data[0] = 0xCB

	err := c.Step(mem, nil)

	if !errors.Is(err, ErrUnimplementedOpcode) {
		t.Fatalf("err=%v, want ErrUnimplementedOpcode", err)
	}
}

func TestDisassembleFormatsOperands(t *testing.T) {
	mem := NewFlatMemory()
	mem.Data[0] = LXI(PairHL)
	mem.Data[1] = 0x34
	mem.Data[2] = 0x12

	d, err := Disassemble(mem, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := d.String(), "0000: 21 34 12 LXI H,#0x1234"; got != want {
		t.Fatalf("disassembly = %q, want %q", got, want)
	}
}

func runUntilStopped(t *testing.T, c *CPU8080, mem Memory, io IO, limit int) {
	t.Helper()
	for i := 0; i < limit; i++ {
		err := c.Step(mem, io)
		if errors.Is(err, ErrCPUStopped) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("CPU did not stop in %d steps", limit)
}
