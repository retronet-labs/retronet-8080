// Comando retronet-8080: runner minimale dell'emulatore Intel 8080.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/retronet-labs/retronet-8080/conformance"
	"github.com/retronet-labs/retronet-8080/cpu"
	"github.com/retronet-labs/retronet-8080/machine"
)

type runConfig struct {
	binPath       string
	profileName   string
	listProfiles  bool
	loadAt        uint16
	startPC       uint16
	steps         uint64
	disasm        uint64
	trace         bool
	ioTrace       bool
	terminal      bool
	terminalInput string
	terminalIn    byte
	terminalOut   byte
	loopbacks     loopbackFlags
	panel         bool
	panelSwitches byte
	panelInputSet bool
	panelAddress  uint16
	ready         bool
	interruptRST  byte
	interruptSet  bool
	traceJSONPath string
	breakPC       addressFlags
	breakOpcode   byteFlags
	watchMemory   addressFlags
	breakInput    byteFlags
	breakOutput   byteFlags
	conformance   bool
	verifyROMPath string
	romSHA256     string
	romSize       int64
	roms          romFlags
	inputs        inputFlags
}

type romSpec struct {
	name string
	path string
}

type romFlags []romSpec

func (r *romFlags) String() string {
	if r == nil || len(*r) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*r))
	for _, spec := range *r {
		parts = append(parts, spec.name+"="+spec.path)
	}
	return strings.Join(parts, ",")
}

func (r *romFlags) Set(value string) error {
	name, path, ok := strings.Cut(value, "=")
	if !ok {
		return errors.New("usa nome=percorso")
	}
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if name == "" || path == "" {
		return errors.New("nome e percorso ROM sono obbligatori")
	}
	*r = append(*r, romSpec{name: name, path: path})
	return nil
}

type inputSpec struct {
	port  byte
	value byte
}

type inputFlags []inputSpec

func (i *inputFlags) String() string {
	if i == nil || len(*i) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*i))
	for _, spec := range *i {
		parts = append(parts, fmt.Sprintf("%d=0x%02X", spec.port, spec.value))
	}
	return strings.Join(parts, ",")
}

type addressFlags []uint16

func (a *addressFlags) String() string {
	parts := make([]string, len(*a))
	for i, value := range *a {
		parts[i] = fmt.Sprintf("0x%04X", value)
	}
	return strings.Join(parts, ",")
}

func (a *addressFlags) Set(value string) error {
	addr, err := parseAddress(value)
	if err != nil {
		return err
	}
	*a = append(*a, addr)
	return nil
}

type byteFlags []byte

func (b *byteFlags) String() string {
	parts := make([]string, len(*b))
	for i, value := range *b {
		parts[i] = fmt.Sprintf("0x%02X", value)
	}
	return strings.Join(parts, ",")
}

type loopbackSpec struct {
	input  byte
	output byte
}

type loopbackFlags []loopbackSpec

func (l *loopbackFlags) String() string {
	parts := make([]string, len(*l))
	for i, spec := range *l {
		parts[i] = fmt.Sprintf("%d=%d", spec.input, spec.output)
	}
	return strings.Join(parts, ",")
}

func (l *loopbackFlags) Set(value string) error {
	inputText, outputText, ok := strings.Cut(value, "=")
	if !ok {
		return errors.New("usa input=output")
	}
	input, err := parsePort(inputText)
	if err != nil {
		return err
	}
	if err := cpu.ValidateInputPort(input); err != nil {
		return err
	}
	output, err := parsePort(outputText)
	if err != nil {
		return err
	}
	if err := cpu.ValidateOutputPort(output); err != nil {
		return err
	}
	*l = append(*l, loopbackSpec{input: input, output: output})
	return nil
}

func (b *byteFlags) Set(value string) error {
	parsed, err := parseByte(value)
	if err != nil {
		return err
	}
	*b = append(*b, parsed)
	return nil
}

func (i *inputFlags) Set(value string) error {
	portText, valueText, ok := strings.Cut(value, "=")
	if !ok {
		return errors.New("usa porta=valore")
	}
	port, err := parsePort(portText)
	if err != nil {
		return err
	}
	if err := cpu.ValidateInputPort(port); err != nil {
		return err
	}
	n, err := strconv.ParseUint(strings.TrimSpace(valueText), 0, 8)
	if err != nil {
		return err
	}
	*i = append(*i, inputSpec{port: port, value: byte(n)})
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "errore: %v\n", err)
		return 2
	}

	if cfg.listProfiles {
		printProfiles(stdout)
		return 0
	}
	if cfg.conformance {
		return runConformance(stdout)
	}
	if cfg.verifyROMPath != "" {
		return runROMVerification(stdout, stderr, cfg)
	}

	profile, ok := machine.Lookup(cfg.profileName)
	if !ok {
		fmt.Fprintf(stderr, "errore profilo: profilo %q non disponibile\n", cfg.profileName)
		return 2
	}

	c := cpu.NewCPU8080()
	baseMemory, err := profile.NewMemory()
	if err != nil {
		fmt.Fprintf(stderr, "errore mappa memoria: %v\n", err)
		return 2
	}
	mem, err := machine.NewObservableMemory(baseMemory)
	if err != nil {
		fmt.Fprintf(stderr, "errore memoria osservabile: %v\n", err)
		return 2
	}
	ports := profile.NewIO()
	peripherals, err := machine.NewPeripheralBus(ports)
	if err != nil {
		fmt.Fprintf(stderr, "errore bus periferiche: %v\n", err)
		return 2
	}

	for _, spec := range cfg.inputs {
		if err := ports.SetInput(spec.port, spec.value); err != nil {
			fmt.Fprintf(stderr, "errore input I/O: %v\n", err)
			return 2
		}
	}
	if cfg.panelInputSet {
		if err := ports.SetInput(machine.TerminalInputPort, cfg.panelSwitches); err != nil {
			fmt.Fprintf(stderr, "errore switch front panel: %v\n", err)
			return 2
		}
	}
	panel, err := machine.NewFrontPanel(c, mem, ports)
	if err != nil {
		fmt.Fprintf(stderr, "errore front panel: %v\n", err)
		return 2
	}
	panel.SetSwitches(cfg.panelSwitches)
	panel.SetAddress(cfg.panelAddress)
	panel.SetReady(cfg.ready)

	var terminal *machine.Terminal
	if cfg.terminal || cfg.terminalInput != "" {
		terminal = machine.NewTerminal(stdout)
		terminal.QueueInputString(cfg.terminalInput)
		if err := terminal.AttachPeripheral(peripherals, "terminal", machine.TerminalConfig{
			InputPort: cfg.terminalIn, OutputPort: cfg.terminalOut,
		}); err != nil {
			fmt.Fprintf(stderr, "errore terminale: %v\n", err)
			return 2
		}
	}
	for i, spec := range cfg.loopbacks {
		register := machine.NewRegisterPeripheral(0)
		if err := register.Attach(peripherals, fmt.Sprintf("loopback-%d", i), spec.input, spec.output); err != nil {
			fmt.Fprintf(stderr, "errore loopback I/O: %v\n", err)
			return 2
		}
	}
	if cfg.ioTrace {
		if err := registerIOTrace(stdout, ports); err != nil {
			fmt.Fprintf(stderr, "errore trace I/O: %v\n", err)
			return 2
		}
	}

	for _, spec := range cfg.roms {
		data, err := os.ReadFile(spec.path)
		if err != nil {
			fmt.Fprintf(stderr, "errore caricamento ROM %s: %v\n", spec.name, err)
			return 1
		}
		if err := profile.LoadROM(mem, spec.name, data); err != nil {
			fmt.Fprintf(stderr, "errore caricamento ROM %s: %v\n", spec.name, err)
			return 1
		}
	}

	loaded := 0
	if cfg.binPath != "" {
		program, err := os.ReadFile(cfg.binPath)
		if err != nil {
			fmt.Fprintf(stderr, "errore caricamento binario: %v\n", err)
			return 1
		}
		if err := machine.LoadBytes(mem, cfg.loadAt, program); err != nil {
			fmt.Fprintf(stderr, "errore caricamento binario: %v\n", err)
			return 1
		}
		loaded = len(program)
	}

	if cfg.disasm > 0 {
		if err := printDisassembly(stdout, mem, cfg.startPC, cfg.disasm); err != nil {
			fmt.Fprintf(stderr, "errore disassembly: %v\n", err)
			return 1
		}
		if cfg.panel {
			printPanel(stdout, panel.Snapshot())
		}
		return 0
	}

	c.PC = cfg.startPC
	if cfg.interruptSet {
		if err := panel.RequestInterrupt(cpu.RST(cfg.interruptRST)); err != nil {
			fmt.Fprintf(stderr, "errore interrupt: %v\n", err)
			return 1
		}
	}

	var trace io.Writer
	if cfg.trace {
		trace = stdout
	}
	debugActive := cfg.traceJSONPath != "" || len(cfg.breakPC) > 0 || len(cfg.breakOpcode) > 0 ||
		len(cfg.watchMemory) > 0 || len(cfg.breakInput) > 0 || len(cfg.breakOutput) > 0
	var executed uint64
	var stopReason string
	if debugActive {
		debugger, err := configureDebugger(panel, mem, ports, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "errore debugger: %v\n", err)
			return 2
		}
		finishTrace, traceErr := configureStructuredTrace(debugger, cfg.traceJSONPath)
		if traceErr != nil {
			fmt.Fprintf(stderr, "errore trace JSON: %v\n", traceErr)
			return 1
		}
		result, runErr := debugger.Run(cfg.steps)
		if finishTrace != nil {
			if traceErr := finishTrace(); runErr == nil && traceErr != nil {
				runErr = traceErr
			}
		}
		executed = result.Steps
		stopReason = string(result.Reason)
		err = runErr
	} else {
		var reason machine.PanelStopReason
		executed, reason, err = runSteps(panel, mem, cfg.steps, trace)
		stopReason = string(reason)
	}
	limitReached := stopReason == string(machine.PanelStoppedByLimit) || stopReason == string(machine.DebugStoppedLimit)
	printDump(stdout, c, cfg, loaded, len(cfg.roms), executed, limitReached, stopReason)
	if cfg.panel {
		printPanel(stdout, panel.Snapshot())
	}
	if terminal != nil && terminal.Err() != nil {
		fmt.Fprintf(stderr, "errore output terminale: %v\n", terminal.Err())
		return 1
	}
	if err != nil {
		fmt.Fprintf(stderr, "errore esecuzione: %v\n", err)
		return 1
	}
	return 0
}

func parseFlags(args []string, stderr io.Writer) (runConfig, error) {
	fs := flag.NewFlagSet("retronet-8080", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfg runConfig
	loadAt := fs.String("addr", "0x0000", "indirizzo di caricamento, decimale o 0xHEX")
	startPC := fs.String("pc", "", "program counter iniziale, default uguale ad -addr")
	panelSwitches := fs.String("panel-switches", "", "valore degli switch dati, decimale o 0xHEX")
	panelAddress := fs.String("panel-address", "", "indirizzo esaminato dal pannello, default uguale al PC iniziale")
	interruptRST := fs.String("interrupt-rst", "", "forza RST 0..7 prima del primo fetch")
	terminalIn := fs.Uint("terminal-in-port", uint(machine.TerminalInputPort), "porta input del terminale")
	terminalOut := fs.Uint("terminal-out-port", uint(machine.TerminalOutputPort), "porta output del terminale")
	fs.StringVar(&cfg.binPath, "bin", "", "percorso della ROM/immagine raw da caricare (es. il .rom prodotto da retronet-asm)")
	fs.StringVar(&cfg.profileName, "profile", "generic", "profilo macchina da usare")
	fs.BoolVar(&cfg.listProfiles, "profiles", false, "elenca i profili macchina disponibili")
	fs.Var(&cfg.roms, "rom", "carica una ROM di profilo nel formato nome=percorso; ripetibile")
	fs.Var(&cfg.inputs, "input", "inizializza una porta input nel formato porta=valore; ripetibile")
	fs.Uint64Var(&cfg.steps, "steps", machine.DefaultStepLimit, "numero massimo di istruzioni da eseguire")
	fs.Uint64Var(&cfg.disasm, "disasm", 0, "disassembla N istruzioni e termina senza eseguire")
	fs.BoolVar(&cfg.trace, "trace", false, "stampa ogni istruzione prima dell'esecuzione")
	fs.BoolVar(&cfg.ioTrace, "io-trace", false, "stampa letture e scritture I/O tramite callback")
	fs.BoolVar(&cfg.terminal, "terminal", false, "collega un terminale ASCII alle porte convenzionali 0/1")
	fs.StringVar(&cfg.terminalInput, "terminal-input", "", "accoda testo ASCII al terminale e abilita -terminal")
	fs.Var(&cfg.loopbacks, "loopback", "registro I/O generico input=output; ripetibile")
	fs.BoolVar(&cfg.panel, "panel", false, "stampa lo stato del front panel dopo l'esecuzione")
	fs.BoolVar(&cfg.ready, "ready", true, "livello READY globale; false ferma il run in WAIT")
	fs.StringVar(&cfg.traceJSONPath, "trace-json", "", "scrive eventi JSON Lines nel file indicato")
	fs.Var(&cfg.breakPC, "break", "breakpoint PC a 16 bit; ripetibile")
	fs.Var(&cfg.breakOpcode, "break-opcode", "breakpoint opcode a 8 bit; ripetibile")
	fs.Var(&cfg.watchMemory, "watch", "watchpoint scrittura memoria a 16 bit; ripetibile")
	fs.Var(&cfg.breakInput, "break-input", "breakpoint porta input; ripetibile")
	fs.Var(&cfg.breakOutput, "break-output", "breakpoint porta output; ripetibile")
	fs.BoolVar(&cfg.conformance, "conformance", false, "esegue la suite sintetica integrata e termina")
	fs.StringVar(&cfg.verifyROMPath, "verify-rom", "", "calcola e verifica una ROM locale senza eseguirla")
	fs.StringVar(&cfg.romSHA256, "rom-sha256", "", "SHA-256 atteso per -verify-rom")
	fs.Int64Var(&cfg.romSize, "rom-size", -1, "dimensione attesa per -verify-rom; -1 la ignora")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.listProfiles {
		return cfg, nil
	}
	if *terminalIn > 0xFF || *terminalOut > 0xFF {
		return cfg, errors.New("porta terminale fuori range")
	}
	cfg.terminalIn = byte(*terminalIn)
	cfg.terminalOut = byte(*terminalOut)
	if err := cpu.ValidateInputPort(cfg.terminalIn); err != nil {
		return cfg, err
	}
	if err := cpu.ValidateOutputPort(cfg.terminalOut); err != nil {
		return cfg, err
	}
	if cfg.conformance || cfg.verifyROMPath != "" {
		return cfg, nil
	}
	if cfg.binPath == "" && len(cfg.roms) == 0 {
		fs.Usage()
		return cfg, errors.New("flag -bin o -rom obbligatorio")
	}

	addr, err := parseAddress(*loadAt)
	if err != nil {
		return cfg, fmt.Errorf("addr non valido: %w", err)
	}
	cfg.loadAt = addr

	if *startPC == "" {
		cfg.startPC = cfg.loadAt
	} else {
		pc, err := parseAddress(*startPC)
		if err != nil {
			return cfg, fmt.Errorf("pc non valido: %w", err)
		}
		cfg.startPC = pc
	}

	if *panelAddress == "" {
		cfg.panelAddress = cfg.startPC
	} else {
		addr, err := parseAddress(*panelAddress)
		if err != nil {
			return cfg, fmt.Errorf("panel-address non valido: %w", err)
		}
		cfg.panelAddress = addr
		cfg.panel = true
	}
	if *panelSwitches != "" {
		value, err := parseByte(*panelSwitches)
		if err != nil {
			return cfg, fmt.Errorf("panel-switches non valido: %w", err)
		}
		cfg.panelSwitches = value
		cfg.panelInputSet = true
		cfg.panel = true
	}
	if *interruptRST != "" {
		value, err := parseByte(*interruptRST)
		if err != nil || value > 7 {
			return cfg, fmt.Errorf("interrupt-rst non valido: usa un vettore 0..7")
		}
		cfg.interruptRST = value
		cfg.interruptSet = true
	}
	return cfg, nil
}

func runConformance(stdout io.Writer) int {
	result := conformance.RunSuite(conformance.SyntheticSuite())
	for _, test := range result.Cases {
		status := "PASS"
		if !test.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(stdout, "%s %s steps=%d stop=%s", status, test.Name, test.Steps, test.StopReason)
		if test.Error != "" {
			fmt.Fprintf(stdout, " error=%s", test.Error)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "conformance passed=%d failed=%d\n", result.Passed, result.Failed)
	if result.Failed > 0 {
		return 1
	}
	return 0
}

func runROMVerification(stdout io.Writer, stderr io.Writer, cfg runConfig) int {
	result, err := conformance.VerifyLocalROM(cfg.verifyROMPath, conformance.ROMExpectation{
		Name:           cfg.verifyROMPath,
		ExpectedSize:   cfg.romSize,
		ExpectedSHA256: cfg.romSHA256,
	})
	if err != nil {
		fmt.Fprintf(stderr, "errore verifica ROM: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "rom path=%s size=%d sha256=%s matches=%v\n",
		result.Path, result.ActualSize, result.ActualSHA256, result.Matches)
	if !result.Matches {
		return 1
	}
	return 0
}

func parseAddress(value string) (uint16, error) {
	value = strings.TrimSpace(value)
	n, err := strconv.ParseUint(value, 0, 16)
	if err != nil {
		return 0, err
	}
	return uint16(n), nil
}

func parsePort(value string) (byte, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(value), 0, 8)
	if err != nil {
		return 0, err
	}
	return byte(n), nil
}

func parseByte(value string) (byte, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(value), 0, 8)
	if err != nil {
		return 0, err
	}
	return byte(n), nil
}

func runSteps(panel *machine.FrontPanel, mem cpu.Memory, limit uint64, trace io.Writer) (uint64, machine.PanelStopReason, error) {
	var observer machine.PanelStepObserver
	if trace != nil {
		observer = func(step uint64, state cpu.CPU8080) error {
			d, err := cpu.Disassemble(mem, state.PC)
			if err != nil {
				return err
			}
			fmt.Fprintf(trace, "trace=%d %s\n", step, d.String())
			return nil
		}
	}
	result, err := panel.Run(limit, observer)
	return result.Steps, result.Reason, err
}

func configureDebugger(panel *machine.FrontPanel, mem cpu.Memory, ports *machine.CallbackIO, cfg runConfig) (*machine.Debugger, error) {
	debugger, err := machine.NewDebugger(panel, mem, ports)
	if err != nil {
		return nil, err
	}
	for _, addr := range cfg.breakPC {
		debugger.AddPCBreakpoint(addr)
	}
	for _, code := range cfg.breakOpcode {
		debugger.AddOpcodeBreakpoint(code)
	}
	for _, addr := range cfg.watchMemory {
		debugger.AddMemoryWatchpoint(addr)
	}
	for _, port := range cfg.breakInput {
		if err := debugger.AddInputBreakpoint(port); err != nil {
			return nil, err
		}
	}
	for _, port := range cfg.breakOutput {
		if err := debugger.AddOutputBreakpoint(port); err != nil {
			return nil, err
		}
	}
	return debugger, nil
}

func configureStructuredTrace(debugger *machine.Debugger, path string) (func() error, error) {
	if path == "" {
		return nil, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	encoder := json.NewEncoder(file)
	var encodeErr error
	debugger.SetTraceSink(func(event machine.TraceEvent) {
		if encodeErr == nil {
			encodeErr = encoder.Encode(event)
		}
	})
	return func() error {
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		return closeErr
	}, nil
}

func printDisassembly(w io.Writer, mem cpu.Memory, pc uint16, count uint64) error {
	for i := uint64(0); i < count; i++ {
		d, err := cpu.Disassemble(mem, pc)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, d.String())
		pc = d.NextPC
	}
	return nil
}

func registerIOTrace(w io.Writer, ioBus *machine.CallbackIO) error {
	for i := 0; i < 256; i++ {
		port := byte(i)
		if err := ioBus.ObserveInput(port, func(port byte, value byte) {
			fmt.Fprintf(w, "io in port=%d value=0x%02X\n", port, value)
		}); err != nil {
			return err
		}
	}
	for i := 0; i < 256; i++ {
		port := byte(i)
		if err := ioBus.ObserveOutput(port, func(port byte, value byte) {
			fmt.Fprintf(w, "io out port=%d value=0x%02X\n", port, value)
		}); err != nil {
			return err
		}
	}
	return nil
}

func printProfiles(w io.Writer) {
	for _, profile := range machine.Profiles() {
		fmt.Fprintf(w, "%s: %s\n", profile.Name, profile.Description)
		if profile.HistoricalNote != "" {
			fmt.Fprintf(w, "  note %s\n", profile.HistoricalNote)
		}
		for _, region := range profile.MemoryRegions {
			fmt.Fprintf(w, "  mem %s 0x%04X-0x%04X %s - %s\n", region.Name, region.Start, region.End, region.Kind, region.Description)
		}
		for _, slot := range profile.ROMSlots {
			required := "optional"
			if slot.Required {
				required = "required"
			}
			fmt.Fprintf(w, "  rom %s @0x%04X max=%d %s - %s\n", slot.Name, slot.Address, slot.MaxSize, required, slot.Description)
		}
		for _, port := range profile.IOPorts {
			historical := "emu"
			if port.Historical {
				historical = "historical"
			}
			fmt.Fprintf(w, "  io %s %d %s %s - %s\n", port.Direction, port.Port, port.Name, historical, port.Description)
		}
		for _, hint := range profile.ROMHints {
			included := "external"
			if hint.Included {
				included = "included"
			}
			fmt.Fprintf(w, "  hint %s slot=%s %s - %s\n", hint.Name, hint.Slot, included, hint.Description)
		}
	}
}

func printDump(w io.Writer, c *cpu.CPU8080, cfg runConfig, loaded int, roms int, executed uint64, limitReached bool, stopReason string) {
	fmt.Fprintf(w, "profile=%s loaded=%d roms=%d addr=0x%04X pc_start=0x%04X steps=%d limit_reached=%v stop_reason=%s\n", cfg.profileName, loaded, roms, cfg.loadAt, cfg.startPC, executed, limitReached, stopReason)
	fmt.Fprintf(w, "A=0x%02X B=0x%02X C=0x%02X D=0x%02X E=0x%02X H=0x%02X L=0x%02X\n", c.A, c.B, c.C, c.D, c.E, c.H, c.L)
	fmt.Fprintf(w, "PC=0x%04X SP=%d Halted=%v Stopped=%v\n", c.PC, c.SP, c.Halted, c.Stopped)
	fmt.Fprintf(w, "Flags C=%v Z=%v S=%v P=%v AC=%v IE=%v\n", c.Carry, c.Zero, c.Sign, c.Parity, c.AuxiliaryCarry, c.InterruptsEnabled)
	fmt.Fprintf(w, "Timing instructions=%d states=%d waits=%d last_states=%d last_waits=%d last_cycles=%s\n",
		c.InstructionCount, c.StateCount, c.WaitStateCount, c.LastTiming.States,
		c.LastTiming.WaitStates, formatCycles(c.LastTiming.MachineCycles()))
}

func formatCycles(cycles []cpu.MachineCycle) string {
	parts := make([]string, len(cycles))
	for i, cycle := range cycles {
		parts[i] = string(cycle)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func printPanel(w io.Writer, state machine.FrontPanelState) {
	fmt.Fprintf(w, "panel address=0x%04X data=0x%02X switches=0x%02X running=%v stop_requested=%v ready=%v waiting=%v interrupt_pending=%v\n",
		state.Address, state.Data, state.Switches, state.Running, state.StopRequested,
		state.Ready, state.Waiting, state.InterruptPending)
}
