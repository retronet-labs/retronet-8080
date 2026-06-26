package cpu

import (
	"testing"

	"github.com/retronet-labs/retronet-hardware/bridge/i8080"
)

// TestGateVsNativeALUDifferential verifica che i due backend aritmetico-logici,
// quello a porte logiche (Gate) e quello con operatori Go (Native), producano lo
// stesso risultato e gli stessi flag su OGNI ingresso e per ogni gruppo. È la
// garanzia che scegliere Native per la velocità non cambi il comportamento.
func TestGateVsNativeALUDifferential(t *testing.T) {
	groups := []byte{
		i8080.GroupADD, i8080.GroupADC, i8080.GroupSUB, i8080.GroupSBB,
		i8080.GroupANA, i8080.GroupXRA, i8080.GroupORA, i8080.GroupCMP,
	}
	for _, g := range groups {
		for a := 0; a <= 0xFF; a++ {
			for v := 0; v <= 0xFF; v++ {
				for _, cy := range []bool{false, true} {
					gr, gf := Gate.ALU(g, byte(a), byte(v), cy)
					nr, nf := Native.ALU(g, byte(a), byte(v), cy)
					if gr != nr || gf != nf {
						t.Fatalf("ALU group=%d a=%#02x v=%#02x cy=%v: gate (%#02x,%+v) != native (%#02x,%+v)",
							g, a, v, cy, gr, gf, nr, nf)
					}
				}
			}
		}
	}
}

// TestGateVsNativeIncDec confronta i due backend su INR/DCR per tutti i valori.
func TestGateVsNativeIncDec(t *testing.T) {
	for v := 0; v <= 0xFF; v++ {
		gr, gf := Gate.Increment(byte(v))
		nr, nf := Native.Increment(byte(v))
		if gr != nr || gf != nf {
			t.Fatalf("Increment(%#02x): gate (%#02x,%+v) != native (%#02x,%+v)", v, gr, gf, nr, nf)
		}
		gr, gf = Gate.Decrement(byte(v))
		nr, nf = Native.Decrement(byte(v))
		if gr != nr || gf != nf {
			t.Fatalf("Decrement(%#02x): gate (%#02x,%+v) != native (%#02x,%+v)", v, gr, gf, nr, nf)
		}
	}
}

// TestGateVsNativeAdd16 confronta DAD su una griglia fitta di parole a 16 bit
// più i casi limite di riporto.
func TestGateVsNativeAdd16(t *testing.T) {
	const step = 0x53 // primo dispari: campiona tutte le classi di riporto
	for a := 0; a <= 0xFFFF; a += step {
		for v := 0; v <= 0xFFFF; v += step {
			gr, gc := Gate.Add16(uint16(a), uint16(v))
			nr, nc := Native.Add16(uint16(a), uint16(v))
			if gr != nr || gc != nc {
				t.Fatalf("Add16(%#04x,%#04x): gate (%#04x,%v) != native (%#04x,%v)", a, v, gr, gc, nr, nc)
			}
		}
	}
	edge := []uint16{0x0000, 0x0001, 0x00FF, 0x0100, 0x7FFF, 0x8000, 0xFFFE, 0xFFFF}
	for _, a := range edge {
		for _, v := range edge {
			gr, gc := Gate.Add16(a, v)
			nr, nc := Native.Add16(a, v)
			if gr != nr || gc != nc {
				t.Fatalf("Add16(%#04x,%#04x): gate (%#04x,%v) != native (%#04x,%v)", a, v, gr, gc, nr, nc)
			}
		}
	}
}
