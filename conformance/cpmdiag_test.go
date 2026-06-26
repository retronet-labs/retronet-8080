package conformance

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/retronet-labs/retronet-8080/cpu"
)

// Questo test valida il core 8080 con una ROM diagnostica CP/M reale
// (es. TST8080.COM, 8080PRE.COM, CPUDIAG.BIN). La ROM NON è versionata: va
// fornita localmente via la variabile d'ambiente RETRONET_8080_DIAG_ROM oppure
// messa in conformance/testdata/diag/ (cartella gitignored). Se assente, il test
// si salta, così la suite resta verde per chi non ha la ROM.

// diagROMPath individua la ROM diagnostica senza versionarla.
func diagROMPath() string {
	if p := os.Getenv("RETRONET_8080_DIAG_ROM"); p != "" {
		return p
	}
	matches, _ := filepath.Glob("testdata/diag/*")
	for _, m := range matches {
		if strings.HasSuffix(m, ".gitignore") {
			continue
		}
		// 8080EXM (esaustiva) è troppo lenta per l'auto-discovery: si lancia solo
		// esplicitamente via RETRONET_8080_DIAG_ROM.
		if strings.Contains(strings.ToUpper(m), "EXM") {
			continue
		}
		if info, err := os.Stat(m); err == nil && !info.IsDir() {
			return m
		}
	}
	return ""
}

// runCPMDiagnostic carica un .com CP/M a 0x0100 e lo esegue, emulando le sole
// chiamate BDOS di stampa usate dalle diagnostiche. Restituisce l'output.
func runCPMDiagnostic(t *testing.T, romPath string) string {
	t.Helper()
	data, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("lettura ROM %q: %v", romPath, err)
	}

	mem := cpu.NewFlatMemory()
	const origin = 0x0100
	for i, b := range data {
		mem.Data[origin+i] = b
	}
	mem.Data[0x0005] = 0xC9 // RET di sicurezza all'entry BDOS

	c := cpu.NewCPU8080()
	// Backend ALU: di default le porte logiche (Gate). RETRONET_8080_ALU=native
	// passa agli operatori Go, molto più veloce per le diagnostiche esaustive
	// (8080EXM) — il comportamento è garantito identico dal test differenziale.
	backend := "gate"
	if strings.EqualFold(os.Getenv("RETRONET_8080_ALU"), "native") {
		c.SetALU(cpu.Native)
		backend = "native"
	}
	t.Logf("backend ALU: %s", backend)
	c.PC = origin
	c.SP = 0xFF00 // pila in cima alla TPA (le ROM di solito la reimpostano)
	io := cpu.NewPorts()

	var out strings.Builder
	maxSteps := 60_000_000 // default; le diagnostiche esaustive (8080EXM) ne servono molti di più
	if s := os.Getenv("RETRONET_8080_DIAG_MAXSTEPS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			maxSteps = n // <= 0 = nessun limite
		}
	}
	for i := 0; maxSteps <= 0 || i < maxSteps; i++ {
		if c.Halted || c.Stopped {
			return out.String()
		}
		switch c.PC {
		case 0x0000:
			return out.String() // warm boot CP/M: fine programma
		case 0x0005:
			bdosPrint(c, mem, &out)
			lo := uint16(mem.Read(c.SP))
			hi := uint16(mem.Read(c.SP + 1))
			c.SP += 2
			c.PC = hi<<8 | lo // RET
			continue
		}
		if err := c.Step(mem, io); err != nil {
			t.Fatalf("errore di esecuzione a PC=0x%04X: %v", c.PC, err)
		}
	}
	t.Fatalf("la ROM non è terminata entro %d step; output parziale:\n%s", maxSteps, out.String())
	return out.String()
}

// bdosPrint emula le funzioni BDOS di stampa: C=2 (carattere in E), C=9 (stringa
// terminata da '$' all'indirizzo in DE).
func bdosPrint(c *cpu.CPU8080, mem *cpu.FlatMemory, out *strings.Builder) {
	emit := func(b byte) {
		out.WriteByte(b)
		os.Stdout.Write([]byte{b}) // streaming live (utile per le corse lunghe)
	}
	switch c.C {
	case 2:
		emit(c.E)
	case 9:
		addr := uint16(c.D)<<8 | uint16(c.E)
		for {
			ch := mem.Read(addr)
			if ch == '$' {
				break
			}
			emit(ch)
			addr++
		}
	}
}

func TestCPMDiagnosticROM(t *testing.T) {
	romPath := diagROMPath()
	if romPath == "" {
		t.Skip("ROM diagnostica non trovata: fornisci RETRONET_8080_DIAG_ROM=/percorso/rom " +
			"oppure metti un .com/.bin in conformance/testdata/diag/ (gitignored).")
	}

	out := runCPMDiagnostic(t, romPath)
	t.Logf("ROM %s — output:\n%s", filepath.Base(romPath), strings.TrimSpace(out))

	if strings.TrimSpace(out) == "" {
		t.Fatalf("nessun output dalla diagnostica")
	}
	if u := strings.ToUpper(out); strings.Contains(u, "FAIL") || strings.Contains(u, "ERROR") {
		t.Fatalf("la diagnostica ha segnalato un errore:\n%s", out)
	}
	// Esito positivo tipico: TST8080/CPUDIAG -> "CPU IS OPERATIONAL";
	// 8080PRE -> "...tests complete".
}
