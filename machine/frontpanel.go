package machine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/retronet-labs/retronet-8080/cpu"
)

var (
	ErrNilCPU               = errors.New("cpu 8080 non inizializzata")
	ErrFrontPanelRunning    = errors.New("front panel in esecuzione")
	ErrInvalidRestartVector = errors.New("vettore RST non valido")
	ErrCPUWaiting           = errors.New("cpu 8080 in stato WAIT")
	ErrInterruptPending     = errors.New("interrupt 8080 gia' pendente")
)

// PanelStopReason descrive perche' un run del front panel e' terminato.
type PanelStopReason string

const (
	PanelStoppedByCPU     PanelStopReason = "cpu-stopped"
	PanelStoppedByRequest PanelStopReason = "requested"
	PanelStoppedByLimit   PanelStopReason = "limit"
	PanelStoppedByReady   PanelStopReason = "waiting"
)

// CycleContext identifica il ciclo macchina sul quale viene campionato READY.
type CycleContext struct {
	PC        uint16
	Opcode    byte
	Mnemonic  string
	Index     byte
	Cycle     cpu.MachineCycle
	Interrupt bool
}

// ReadyCallback decide se uno specifico ciclo puo' raggiungere T3.
type ReadyCallback func(CycleContext) bool

// PanelRunResult riassume un'esecuzione sincrona del front panel.
type PanelRunResult struct {
	Steps  uint64
	Reason PanelStopReason
}

// PanelStepObserver viene chiamato prima di ogni istruzione con una copia
// dello stato CPU, utile per trace e debugger senza esporre mutazioni.
type PanelStepObserver func(step uint64, state cpu.CPU8080) error

// FrontPanelState e' una fotografia delle luci e dei selettori modellati.
type FrontPanelState struct {
	CPU              cpu.CPU8080
	Switches         byte
	Address          uint16
	Data             byte
	Running          bool
	StopRequested    bool
	Ready            bool
	Waiting          bool
	WaitCycle        CycleContext
	InterruptPending bool
}

type interruptRequest struct {
	code         byte
	operands     [2]byte
	operandCount byte
}

// JamInstruction e' una copia pubblica di una richiesta interrupt pendente.
type JamInstruction struct {
	Code         byte
	Operands     [2]byte
	OperandCount byte
}

// FrontPanel coordina il core, la memoria e l'I/O come dispositivo esterno.
// Solo Stop e i selettori atomici sono pensati per uso concorrente con Run.
type FrontPanel struct {
	cpu    *cpu.CPU8080
	memory cpu.Memory
	io     cpu.IO

	switches      atomic.Uint32
	address       atomic.Uint32
	running       atomic.Bool
	stopRequested atomic.Bool
	ready         atomic.Bool
	waiting       atomic.Bool

	readyCallback   ReadyCallback
	cycleActive     bool
	cycleIndex      byte
	activeOpcode    cpu.Opcode
	activeInterrupt bool
	activeRequest   interruptRequest
	waitCycle       CycleContext

	interruptMu      sync.Mutex
	pendingInterrupt *interruptRequest
}

// NewFrontPanel crea un pannello sopra componenti gia' configurati.
func NewFrontPanel(c *cpu.CPU8080, memory cpu.Memory, ioBus cpu.IO) (*FrontPanel, error) {
	if c == nil {
		return nil, ErrNilCPU
	}
	if memory == nil {
		return nil, cpu.ErrNilMemory
	}
	panel := &FrontPanel{cpu: c, memory: memory, io: ioBus}
	panel.ready.Store(true)
	return panel, nil
}

// SetSwitches imposta gli otto switch dati.
func (p *FrontPanel) SetSwitches(value byte) {
	p.switches.Store(uint32(value))
}

// Switches legge gli otto switch dati.
func (p *FrontPanel) Switches() byte {
	return byte(p.switches.Load())
}

// SetAddress imposta i quattordici switch indirizzo.
func (p *FrontPanel) SetAddress(addr uint16) {
	p.address.Store(uint32(addr & cpu.AddressMask))
}

// Address legge l'indirizzo selezionato.
func (p *FrontPanel) Address() uint16 {
	return uint16(p.address.Load())
}

// Examine legge la memoria all'indirizzo selezionato.
func (p *FrontPanel) Examine() byte {
	return p.memory.Read(p.Address())
}

// Deposit scrive un byte all'indirizzo selezionato. Il bus mantiene la propria
// policy: per esempio una MemoryBus ignora una scrittura in ROM.
func (p *FrontPanel) Deposit(value byte) error {
	if p.running.Load() {
		return ErrFrontPanelRunning
	}
	p.memory.Write(p.Address(), value)
	return nil
}

// DepositSwitches deposita il valore degli switch dati.
func (p *FrontPanel) DepositSwitches() error {
	return p.Deposit(p.Switches())
}

// AttachSwitches collega gli switch dati a una porta input callback.
func (p *FrontPanel) AttachSwitches(ioBus *CallbackIO, port byte) error {
	if ioBus == nil {
		return cpu.ErrNilIO
	}
	return ioBus.OnInput(port, func(_ byte, _ byte) byte {
		return p.Switches()
	})
}

// SetReady imposta il livello READY globale usato senza callback specifica.
func (p *FrontPanel) SetReady(ready bool) {
	p.ready.Store(ready)
}

// Ready restituisce il livello READY globale.
func (p *FrontPanel) Ready() bool {
	return p.ready.Load()
}

// SetReadyCallback collega una policy READY per ciclo macchina.
func (p *FrontPanel) SetReadyCallback(callback ReadyCallback) error {
	if p.running.Load() {
		return ErrFrontPanelRunning
	}
	p.readyCallback = callback
	return nil
}

// RequestInterrupt accoda una istruzione da forzare al prossimo confine PCI.
func (p *FrontPanel) RequestInterrupt(code byte, operands ...byte) error {
	op := cpu.Decode(code)
	want := int(op.Length - 1)
	if len(operands) != want {
		return fmt.Errorf("%w: opcode=0x%02X operands=%d want=%d", cpu.ErrInvalidJamInstruction, code, len(operands), want)
	}
	request := &interruptRequest{code: code, operandCount: byte(want)}
	copy(request.operands[:], operands)

	p.interruptMu.Lock()
	defer p.interruptMu.Unlock()
	if p.pendingInterrupt != nil {
		return ErrInterruptPending
	}
	p.pendingInterrupt = request
	return nil
}

// InterruptPending indica se una jam attende il prossimo confine PCI.
func (p *FrontPanel) InterruptPending() bool {
	p.interruptMu.Lock()
	defer p.interruptMu.Unlock()
	return p.pendingInterrupt != nil
}

// PendingInterrupt restituisce la jam instruction in attesa.
func (p *FrontPanel) PendingInterrupt() (JamInstruction, bool) {
	p.interruptMu.Lock()
	defer p.interruptMu.Unlock()
	if p.pendingInterrupt == nil {
		return JamInstruction{}, false
	}
	return JamInstruction{
		Code:         p.pendingInterrupt.code,
		Operands:     p.pendingInterrupt.operands,
		OperandCount: p.pendingInterrupt.operandCount,
	}, true
}

// Reset applica il reset storico della CPU senza modificare i selettori.
func (p *FrontPanel) Reset() error {
	if p.running.Load() {
		return ErrFrontPanelRunning
	}
	p.stopRequested.Store(false)
	p.clearCycleState()
	p.interruptMu.Lock()
	p.pendingInterrupt = nil
	p.interruptMu.Unlock()
	p.cpu.Reset()
	return nil
}

// Jam forza una istruzione esterna, come il circuito di interrupt dell'8080.
func (p *FrontPanel) Jam(code byte, operands ...byte) error {
	if p.running.Load() {
		return ErrFrontPanelRunning
	}
	p.clearCycleState()
	return p.cpu.Jam(p.memory, p.io, code, operands...)
}

// InterruptRST forza un restart vettorizzato da 0 a 7.
func (p *FrontPanel) InterruptRST(vector byte) error {
	if vector > 7 {
		return fmt.Errorf("%w: %d", ErrInvalidRestartVector, vector)
	}
	return p.Jam(cpu.RST(vector))
}

// Step esegue una singola istruzione se la CPU e' in stato running.
func (p *FrontPanel) Step() error {
	if err := p.prepareCycles(); err != nil {
		return err
	}
	for p.cycleIndex < p.activeOpcode.CycleCount {
		context := CycleContext{
			PC:        p.cpu.PC,
			Opcode:    p.activeOpcode.Code,
			Mnemonic:  p.activeOpcode.Mnemonic,
			Index:     p.cycleIndex,
			Cycle:     p.activeOpcode.Cycles[p.cycleIndex],
			Interrupt: p.activeInterrupt,
		}
		ready := p.ready.Load()
		if p.readyCallback != nil {
			ready = p.readyCallback(context)
		}
		if !ready {
			p.waitCycle = context
			p.waiting.Store(true)
			p.cpu.RecordWaitState()
			return ErrCPUWaiting
		}
		p.cycleIndex++
	}

	interrupt := p.activeInterrupt
	request := p.activeRequest
	p.clearCycleState()
	if !interrupt {
		return p.cpu.Step(p.memory, p.io)
	}
	operands := request.operands[:request.operandCount]
	if err := p.cpu.Jam(p.memory, p.io, request.code, operands...); err != nil {
		return err
	}
	p.interruptMu.Lock()
	p.pendingInterrupt = nil
	p.interruptMu.Unlock()
	return nil
}

func (p *FrontPanel) prepareCycles() error {
	if p.cycleActive {
		return nil
	}
	p.interruptMu.Lock()
	if p.pendingInterrupt != nil {
		p.activeRequest = *p.pendingInterrupt
		p.activeOpcode = cpu.Decode(p.pendingInterrupt.code)
		p.activeInterrupt = true
	}
	p.interruptMu.Unlock()
	if !p.activeInterrupt {
		if p.cpu.Halted || p.cpu.Stopped {
			return cpu.ErrCPUStopped
		}
		p.activeOpcode = cpu.Decode(p.memory.Read(p.cpu.PC))
	}
	p.cycleActive = true
	p.cycleIndex = 0
	return nil
}

func (p *FrontPanel) clearCycleState() {
	p.cycleActive = false
	p.cycleIndex = 0
	p.activeOpcode = cpu.Opcode{}
	p.activeInterrupt = false
	p.activeRequest = interruptRequest{}
	p.waiting.Store(false)
	p.waitCycle = CycleContext{}
}

// Stop richiede in modo concorrente l'arresto del prossimo ciclo Run.
func (p *FrontPanel) Stop() {
	p.stopRequested.Store(true)
}

// Run esegue fino a HLT/stopped, richiesta esterna o limite istruzioni.
func (p *FrontPanel) Run(limit uint64, observer PanelStepObserver) (PanelRunResult, error) {
	if !p.running.CompareAndSwap(false, true) {
		return PanelRunResult{}, ErrFrontPanelRunning
	}
	defer p.running.Store(false)

	var result PanelRunResult
	for result.Steps < limit {
		if p.stopRequested.Swap(false) {
			result.Reason = PanelStoppedByRequest
			return result, nil
		}
		if (p.cpu.Halted || p.cpu.Stopped) && !p.InterruptPending() && !p.cycleActive {
			result.Reason = PanelStoppedByCPU
			return result, nil
		}
		if observer != nil {
			if err := observer(result.Steps, *p.cpu); err != nil {
				return result, err
			}
		}
		if err := p.Step(); err != nil {
			if errors.Is(err, ErrCPUWaiting) {
				result.Reason = PanelStoppedByReady
				return result, nil
			}
			if errors.Is(err, cpu.ErrCPUStopped) {
				result.Reason = PanelStoppedByCPU
				return result, nil
			}
			return result, err
		}
		result.Steps++
	}

	if p.cpu.Halted || p.cpu.Stopped {
		result.Reason = PanelStoppedByCPU
	} else if p.stopRequested.Swap(false) {
		result.Reason = PanelStoppedByRequest
	} else {
		result.Reason = PanelStoppedByLimit
	}
	return result, nil
}

// Snapshot fotografa CPU, selettori e byte memoria selezionato.
func (p *FrontPanel) Snapshot() FrontPanelState {
	address := p.Address()
	return FrontPanelState{
		CPU:              *p.cpu,
		Switches:         p.Switches(),
		Address:          address,
		Data:             p.memory.Read(address),
		Running:          p.running.Load(),
		StopRequested:    p.stopRequested.Load(),
		Ready:            p.ready.Load(),
		Waiting:          p.waiting.Load(),
		WaitCycle:        p.waitCycle,
		InterruptPending: p.InterruptPending(),
	}
}
