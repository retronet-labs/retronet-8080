package cpu

// AddressMask copre l'intero spazio indirizzabile dall'Intel 8080.
const AddressMask uint16 = 0xFFFF

// Register identifica i registri nel campo a tre bit degli opcode 8080.
type Register byte

const (
	RegB Register = iota
	RegC
	RegD
	RegE
	RegH
	RegL
	RegM
	RegA
)

// RegisterPair identifica le coppie usate da LXI/INX/DCX/DAD e PUSH/POP.
type RegisterPair byte

const (
	PairBC RegisterPair = iota
	PairDE
	PairHL
	PairSP
)

// Condition identifica le otto condizioni del control flow 8080.
type Condition byte

const (
	CondNZ Condition = iota
	CondZ
	CondNC
	CondC
	CondPO
	CondPE
	CondP
	CondM
)
