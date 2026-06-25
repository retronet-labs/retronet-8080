package conformance

import (
	"os"
	"path/filepath"
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
	c.PC = origin
	c.SP = 0xFF00 // pila in cima alla TPA (le ROM di solito la reimpostano)
	io := cpu.NewPorts()

	var out strings.Builder
	const maxSteps = 60_000_000
	for i := 0; i < maxSteps; i++ {
		if c.Halted || c.Stopped {
			break
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
	t.Fatalf("la ROM non è terminata entro %d step", maxSteps)
	return out.String()
}

// bdosPrint emula le funzioni BDOS di stampa: C=2 (carattere in E), C=9 (stringa
// terminata da '$' all'indirizzo in DE).
func bdosPrint(c *cpu.CPU8080, mem *cpu.FlatMemory, out *strings.Builder) {
	switch c.C {
	case 2:
		out.WriteByte(c.E)
	case 9:
		addr := uint16(c.D)<<8 | uint16(c.E)
		for {
			ch := mem.Read(addr)
			if ch == '$' {
				break
			}
			out.WriteByte(ch)
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
