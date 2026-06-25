package machine

import (
	"errors"
	"fmt"
	"sync"

	"github.com/retronet-labs/retronet-8080/cpu"
)

// TraceEventKind classifica gli eventi strutturati del debugger.
type TraceEventKind string

const (
	TraceInstruction TraceEventKind = "instruction"
	TraceWait        TraceEventKind = "wait"
	TraceBreakpoint  TraceEventKind = "breakpoint"
)

// DebugStopReason descrive il motivo di arresto del debugger.
type DebugStopReason string

const (
	DebugContinue          DebugStopReason = ""
	DebugStoppedCPU        DebugStopReason = "cpu-stopped"
	DebugStoppedRequested  DebugStopReason = "requested"
	DebugStoppedWaiting    DebugStopReason = "waiting"
	DebugStoppedBreakpoint DebugStopReason = "breakpoint"
	DebugStoppedWatchpoint DebugStopReason = "watchpoint"
	DebugStoppedIO         DebugStopReason = "io-breakpoint"
	DebugStoppedLimit      DebugStopReason = "limit"
)

// IOAccess descrive un trasferimento osservato sul bus I/O.
type IOAccess struct {
	Direction IODirection `json:"direction"`
	Port      byte        `json:"port"`
	Value     byte        `json:"value"`
}

// TraceEvent contiene stato prima/dopo e side-effect di una istruzione.
type TraceEvent struct {
	Sequence     uint64                `json:"sequence"`
	Kind         TraceEventKind        `json:"kind"`
	PC           uint16                `json:"pc"`
	Opcode       byte                  `json:"opcode"`
	Bytes        []byte                `json:"bytes,omitempty"`
	Disassembly  string                `json:"disassembly,omitempty"`
	Interrupt    bool                  `json:"interrupt,omitempty"`
	Before       cpu.CPU8080           `json:"before"`
	After        cpu.CPU8080           `json:"after"`
	Timing       cpu.InstructionTiming `json:"timing"`
	MemoryWrites []MemoryWrite         `json:"memory_writes,omitempty"`
	IO           []IOAccess            `json:"io,omitempty"`
	WaitCycle    *CycleContext         `json:"wait_cycle,omitempty"`
	Breakpoint   string                `json:"breakpoint,omitempty"`
}

// DebugRunResult riassume un run controllato dal debugger.
type DebugRunResult struct {
	Steps  uint64
	Reason DebugStopReason
}

// TraceSink riceve ogni evento nell'ordine di esecuzione.
type TraceSink func(TraceEvent)

// Debugger aggiunge trace, breakpoint e watchpoint sopra FrontPanel.
type Debugger struct {
	panel  *FrontPanel
	memory cpu.Memory
	ioBus  *CallbackIO

	pcBreakpoints     map[uint16]struct{}
	opcodeBreakpoints map[byte]struct{}
	memoryWatchpoints map[uint16]struct{}
	inputBreakpoints  [256]bool
	outputBreakpoints [256]bool

	mu           sync.Mutex
	memoryWrites []MemoryWrite
	ioAccesses   []IOAccess
	sequence     uint64
	sink         TraceSink
}

// NewDebugger crea un debugger. Per watchpoint di memoria completi, memory deve
// essere lo stesso ObservableMemory usato dal FrontPanel.
func NewDebugger(panel *FrontPanel, memory cpu.Memory, ioBus *CallbackIO) (*Debugger, error) {
	if panel == nil {
		return nil, errors.New("front panel non inizializzato")
	}
	if memory == nil {
		return nil, cpu.ErrNilMemory
	}
	d := &Debugger{
		panel:             panel,
		memory:            memory,
		ioBus:             ioBus,
		pcBreakpoints:     make(map[uint16]struct{}),
		opcodeBreakpoints: make(map[byte]struct{}),
		memoryWatchpoints: make(map[uint16]struct{}),
	}
	if observed, ok := memory.(*ObservableMemory); ok {
		observed.ObserveWrites(d.recordMemoryWrite)
	}
	if ioBus != nil {
		for i := 0; i < 256; i++ {
			port := byte(i)
			if err := ioBus.ObserveInput(port, d.recordInput); err != nil {
				return nil, err
			}
		}
		for i := 0; i < 256; i++ {
			port := byte(i)
			if err := ioBus.ObserveOutput(port, d.recordOutput); err != nil {
				return nil, err
			}
		}
	}
	return d, nil
}

func (d *Debugger) SetTraceSink(sink TraceSink) { d.sink = sink }

func (d *Debugger) AddPCBreakpoint(addr uint16) {
	d.pcBreakpoints[addr&cpu.AddressMask] = struct{}{}
}

func (d *Debugger) AddOpcodeBreakpoint(code byte) {
	d.opcodeBreakpoints[code] = struct{}{}
}

func (d *Debugger) AddMemoryWatchpoint(addr uint16) {
	d.memoryWatchpoints[addr&cpu.AddressMask] = struct{}{}
}

func (d *Debugger) AddInputBreakpoint(port byte) error {
	if err := cpu.ValidateInputPort(port); err != nil {
		return err
	}
	d.inputBreakpoints[port] = true
	return nil
}

func (d *Debugger) AddOutputBreakpoint(port byte) error {
	if err := cpu.ValidateOutputPort(port); err != nil {
		return err
	}
	d.outputBreakpoints[port] = true
	return nil
}

// Step esegue al massimo una istruzione e produce un evento strutturato.
func (d *Debugger) Step() (TraceEvent, DebugStopReason, error) {
	d.clearAccesses()
	before := d.panel.Snapshot()
	event, err := d.nextEvent(before)
	if err != nil {
		return TraceEvent{}, DebugContinue, err
	}
	if reason := d.breakpointReason(event); reason != "" {
		event.Kind = TraceBreakpoint
		event.Breakpoint = reason
		event.After = before.CPU
		d.emit(&event)
		return event, DebugStoppedBreakpoint, nil
	}

	result, err := d.panel.Run(1, nil)
	if err != nil {
		return TraceEvent{}, DebugContinue, err
	}
	if result.Steps == 0 {
		switch result.Reason {
		case PanelStoppedByReady:
			state := d.panel.Snapshot()
			event.Kind = TraceWait
			event.After = state.CPU
			context := state.WaitCycle
			event.WaitCycle = &context
			d.emit(&event)
			return event, DebugStoppedWaiting, nil
		case PanelStoppedByRequest:
			return TraceEvent{}, DebugStoppedRequested, nil
		default:
			return TraceEvent{}, DebugStoppedCPU, nil
		}
	}

	after := d.panel.Snapshot()
	event.Kind = TraceInstruction
	event.After = after.CPU
	event.Timing = after.CPU.LastTiming
	event.MemoryWrites, event.IO = d.takeAccesses()
	d.emit(&event)

	if d.hitMemoryWatchpoint(event.MemoryWrites) {
		return event, DebugStoppedWatchpoint, nil
	}
	if d.hitIOBreakpoint(event.IO) {
		return event, DebugStoppedIO, nil
	}
	if result.Reason == PanelStoppedByCPU {
		return event, DebugStoppedCPU, nil
	}
	return event, DebugContinue, nil
}

// Run esegue fino a breakpoint, watchpoint, WAIT, HLT o limite.
func (d *Debugger) Run(limit uint64) (DebugRunResult, error) {
	var result DebugRunResult
	for result.Steps < limit {
		event, reason, err := d.Step()
		if err != nil {
			return result, err
		}
		if event.Kind == TraceInstruction {
			result.Steps++
		}
		if reason != DebugContinue {
			result.Reason = reason
			return result, nil
		}
	}
	result.Reason = DebugStoppedLimit
	return result, nil
}

func (d *Debugger) nextEvent(before FrontPanelState) (TraceEvent, error) {
	event := TraceEvent{PC: before.CPU.PC, Before: before.CPU}
	if pending, ok := d.panel.PendingInterrupt(); ok {
		op := cpu.Decode(pending.Code)
		event.Opcode = pending.Code
		event.Interrupt = true
		event.Bytes = append([]byte{pending.Code}, pending.Operands[:pending.OperandCount]...)
		event.Disassembly = fmt.Sprintf("%04X: %02X       %s [interrupt]", event.PC, pending.Code, op.Mnemonic)
		return event, nil
	}
	if before.CPU.Halted || before.CPU.Stopped {
		return event, nil
	}
	disassembly, err := cpu.Disassemble(d.memory, event.PC)
	if err != nil {
		return TraceEvent{}, err
	}
	event.Opcode = disassembly.Opcode.Code
	event.Bytes = append([]byte(nil), disassembly.Bytes[:disassembly.Length]...)
	event.Disassembly = disassembly.String()
	return event, nil
}

func (d *Debugger) breakpointReason(event TraceEvent) string {
	if _, ok := d.pcBreakpoints[event.PC]; ok {
		return fmt.Sprintf("pc=0x%04X", event.PC)
	}
	if len(event.Bytes) > 0 {
		if _, ok := d.opcodeBreakpoints[event.Opcode]; ok {
			return fmt.Sprintf("opcode=0x%02X", event.Opcode)
		}
	}
	return ""
}

func (d *Debugger) hitMemoryWatchpoint(writes []MemoryWrite) bool {
	for _, write := range writes {
		if _, ok := d.memoryWatchpoints[write.Address]; ok {
			return true
		}
	}
	return false
}

func (d *Debugger) hitIOBreakpoint(accesses []IOAccess) bool {
	for _, access := range accesses {
		if access.Direction == IODirectionInput && d.inputBreakpoints[access.Port] {
			return true
		}
		if access.Direction == IODirectionOutput && d.outputBreakpoints[access.Port] {
			return true
		}
	}
	return false
}

func (d *Debugger) recordMemoryWrite(write MemoryWrite) {
	d.mu.Lock()
	d.memoryWrites = append(d.memoryWrites, write)
	d.mu.Unlock()
}

func (d *Debugger) recordInput(port byte, value byte) {
	d.recordIO(IOAccess{Direction: IODirectionInput, Port: port, Value: value})
}

func (d *Debugger) recordOutput(port byte, value byte) {
	d.recordIO(IOAccess{Direction: IODirectionOutput, Port: port, Value: value})
}

func (d *Debugger) recordIO(access IOAccess) {
	d.mu.Lock()
	d.ioAccesses = append(d.ioAccesses, access)
	d.mu.Unlock()
}

func (d *Debugger) clearAccesses() {
	d.mu.Lock()
	d.memoryWrites = d.memoryWrites[:0]
	d.ioAccesses = d.ioAccesses[:0]
	d.mu.Unlock()
}

func (d *Debugger) takeAccesses() ([]MemoryWrite, []IOAccess) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]MemoryWrite(nil), d.memoryWrites...), append([]IOAccess(nil), d.ioAccesses...)
}

func (d *Debugger) emit(event *TraceEvent) {
	event.Sequence = d.sequence
	d.sequence++
	if d.sink != nil {
		d.sink(*event)
	}
}
