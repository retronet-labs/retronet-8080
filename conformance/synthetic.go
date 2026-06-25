package conformance

import (
	"fmt"

	"github.com/retronet-labs/retronet-8080/cpu"
	"github.com/retronet-labs/retronet-8080/machine"
)

// SyntheticSuite restituisce casi indipendenti da ROM e profili storici.
func SyntheticSuite() []Case {
	return []Case{
		{
			Name:    "load-move",
			Program: []byte{cpu.MVI(cpu.RegA), 0x2A, cpu.MOV(cpu.RegB, cpu.RegA), cpu.HLT()},
			Verify: func(ctx *Context, run machine.DebugRunResult) error {
				if ctx.CPU.A != 0x2A || ctx.CPU.B != 0x2A || run.Reason != machine.DebugStoppedCPU {
					return fmt.Errorf("A=0x%02X B=0x%02X stop=%s", ctx.CPU.A, ctx.CPU.B, run.Reason)
				}
				return nil
			},
		},
		{
			Name:    "alu-flags",
			Program: []byte{cpu.MVI(cpu.RegA), 0xFF, cpu.ADI(), 0x01, cpu.HLT()},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.A != 0 || !ctx.CPU.Carry || !ctx.CPU.Zero || ctx.CPU.Sign || !ctx.CPU.Parity || !ctx.CPU.AuxiliaryCarry {
					return fmt.Errorf("A=0x%02X C=%v Z=%v S=%v P=%v AC=%v",
						ctx.CPU.A, ctx.CPU.Carry, ctx.CPU.Zero, ctx.CPU.Sign, ctx.CPU.Parity, ctx.CPU.AuxiliaryCarry)
				}
				return nil
			},
		},
		{
			Name: "memory-indirect",
			Program: []byte{
				cpu.LXI(cpu.PairHL), 0x00, 0x01,
				cpu.MVI(cpu.RegM), 0xA5,
				cpu.MOV(cpu.RegA, cpu.RegM),
				cpu.HLT(),
			},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.A != 0xA5 || ctx.Memory.Read(0x0100) != 0xA5 {
					return fmt.Errorf("A=0x%02X M=0x%02X", ctx.CPU.A, ctx.Memory.Read(0x0100))
				}
				return nil
			},
		},
		{
			Name: "call-return",
			Program: []byte{
				cpu.CALL(), 0x06, 0x00,
				cpu.HLT(), 0x00, 0x00,
				cpu.MVI(cpu.RegA), 0x42,
				cpu.RET(),
			},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.A != 0x42 || ctx.CPU.SP != 0 || ctx.CPU.PC != 0x0004 {
					return fmt.Errorf("A=0x%02X SP=0x%04X PC=0x%04X", ctx.CPU.A, ctx.CPU.SP, ctx.CPU.PC)
				}
				return nil
			},
		},
		{
			Name:    "conditional-jump-taken",
			Program: []byte{cpu.J(cpu.CondNC), 0x06, 0x00, cpu.HLT(), 0x00, 0x00, cpu.HLT()},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.PC != 0x0007 {
					return fmt.Errorf("PC=0x%04X", ctx.CPU.PC)
				}
				return nil
			},
		},
		{
			Name:    "conditional-call-not-taken-timing",
			Program: []byte{cpu.C(cpu.CondC), 0x06, 0x00, cpu.HLT(), 0x00, 0x00, cpu.HLT()},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.StateCount != 18 {
					return fmt.Errorf("StateCount=%d want=18", ctx.CPU.StateCount)
				}
				return nil
			},
		},
		{
			Name:    "rotate-carry",
			Program: []byte{cpu.MVI(cpu.RegA), 0x81, cpu.RLC(), cpu.HLT()},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				if ctx.CPU.A != 0x03 || !ctx.CPU.Carry {
					return fmt.Errorf("A=0x%02X C=%v", ctx.CPU.A, ctx.CPU.Carry)
				}
				return nil
			},
		},
		{
			Name:    "io-echo",
			Program: []byte{cpu.IN(), 0x00, cpu.OUT(), machine.TerminalOutputPort, cpu.HLT()},
			Setup: func(ctx *Context) error {
				return ctx.IO.SetInput(0, 0x5A)
			},
			Verify: func(ctx *Context, _ machine.DebugRunResult) error {
				output, err := ctx.IO.OutputValue(machine.TerminalOutputPort)
				if err != nil {
					return err
				}
				if ctx.CPU.A != 0x5A || output != 0x5A {
					return fmt.Errorf("A=0x%02X OUT=0x%02X", ctx.CPU.A, output)
				}
				return nil
			},
		},
		{
			Name:      "rst-ring",
			Program:   restartRing(),
			StepLimit: 8,
			Verify: func(ctx *Context, run machine.DebugRunResult) error {
				if run.Reason != machine.DebugStoppedLimit || ctx.CPU.SP != 0xFFF0 {
					return fmt.Errorf("stop=%s SP=0x%04X", run.Reason, ctx.CPU.SP)
				}
				return nil
			},
		},
		{
			Name:    "interrupt-rst",
			Program: interruptProgram(),
			Setup: func(ctx *Context) error {
				return ctx.Panel.RequestInterrupt(cpu.RST(1))
			},
			Verify: func(ctx *Context, run machine.DebugRunResult) error {
				if run.Steps != 2 || ctx.CPU.SP != 0xFFFE || ctx.Memory.Read(0xFFFE) != 0x00 {
					return fmt.Errorf("steps=%d SP=0x%04X return-low=0x%02X", run.Steps, ctx.CPU.SP, ctx.Memory.Read(0xFFFE))
				}
				return nil
			},
		},
		{
			Name:    "ready-wait",
			Program: []byte{cpu.NOP()},
			Setup: func(ctx *Context) error {
				ctx.Panel.SetReady(false)
				return nil
			},
			Verify: func(ctx *Context, run machine.DebugRunResult) error {
				if run.Reason != machine.DebugStoppedWaiting || run.Steps != 0 || ctx.CPU.WaitStateCount != 1 {
					return fmt.Errorf("stop=%s steps=%d waits=%d", run.Reason, run.Steps, ctx.CPU.WaitStateCount)
				}
				return nil
			},
		},
	}
}

func restartRing() []byte {
	program := make([]byte, 64)
	for vector := byte(0); vector < 8; vector++ {
		program[int(vector)*8] = cpu.RST((vector + 1) & 0x07)
	}
	return program
}

func interruptProgram() []byte {
	program := make([]byte, 9)
	program[0] = cpu.NOP()
	program[8] = cpu.HLT()
	return program
}
