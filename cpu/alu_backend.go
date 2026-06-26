package cpu

import "github.com/retronet-labs/retronet-hardware/bridge/i8080"

// ALUBackend astrae il motore aritmetico-logico dell'8080. RetroNet ne offre due
// implementazioni intercambiabili con semantica identica, flag compresi
// (Carry, Zero, Sign, Parity e Auxiliary Carry):
//
//   - Gate: l'ALU costruita dalle sole porte logiche di retronet-logic, raggiunta
//     tramite il bridge i8080. È il default e dimostra che la CPU calcola su un
//     datapath fatto di gate.
//   - Native: la stessa semantica espressa con gli operatori aritmetici di Go.
//     È molto più veloce (utile per le diagnostiche esaustive come 8080EXM e per
//     la CI) e serve da oracolo per il test differenziale gate <-> native.
//
// I due backend devono restituire lo stesso risultato e gli stessi flag su ogni
// ingresso: è ciò che verifica TestGateVsNativeALUDifferential.
type ALUBackend interface {
	// ALU esegue uno degli otto gruppi 8080 (ADD/ADC/SUB/SBB/ANA/XRA/ORA/CMP)
	// su A=a e value, con il carry entrante carryIn.
	ALU(group, a, value byte, carryIn bool) (byte, i8080.Flags)
	// Increment esegue value+1 con la semantica di flag di INR (Carry invariato).
	Increment(value byte) (byte, i8080.Flags)
	// Decrement esegue value-1 con la semantica di flag di DCR (Carry invariato).
	Decrement(value byte) (byte, i8080.Flags)
	// Add16 somma due parole a 16 bit (usata da DAD) restituendo il Carry uscente.
	Add16(a, value uint16) (uint16, bool)
}

// Gate è il backend aritmetico-logico costruito dalle porte logiche (default).
var Gate ALUBackend = gateBackend{}

// Native è il backend aritmetico-logico con operatori Go: veloce e oracolo del
// test differenziale verso Gate.
var Native ALUBackend = nativeBackend{}

// gateBackend inoltra ogni operazione all'ALU a porte tramite il bridge i8080.
type gateBackend struct{}

func (gateBackend) ALU(group, a, value byte, carryIn bool) (byte, i8080.Flags) {
	return i8080.ALU(group, a, value, carryIn)
}

func (gateBackend) Increment(value byte) (byte, i8080.Flags) { return i8080.Increment(value) }

func (gateBackend) Decrement(value byte) (byte, i8080.Flags) { return i8080.Decrement(value) }

func (gateBackend) Add16(a, value uint16) (uint16, bool) { return i8080.Add16(a, value) }
