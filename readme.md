# retronet-8080 - Emulatore Intel 8080

Emulatore Intel 8080 scritto in Go, parte dell'ecosistema RetroNet. Il progetto
segue la filosofia dei moduli 4004 e 8008: core importabile, CLI eseguibile,
test, conformance sintetica e documentazione in italiano.

La v0.1 resta volutamente un emulatore 8080: non include CP/M, BDOS/BIOS, dischi
o ROM storiche redistribuite. Queste parti vivranno in moduli successivi.

## Quick Start

```bash
go test ./...
go run ./cmd/retronet-8080 -conformance
go run ./cmd/retronet-8080 -bin programma.bin -steps 1000
go run ./cmd/retronet-8080 -bin programma.bin -disasm 8
go run ./cmd/retronet-8080 -profiles
```

Esempio raw minimo: `MVI A,0x2A; HLT`.

```powershell
[IO.File]::WriteAllBytes("$env:TEMP\load-a.bin", [byte[]](0x3E, 0x2A, 0x76))
go run ./cmd/retronet-8080 -bin "$env:TEMP\load-a.bin" -trace -steps 8
```

## Stato

- CPU 8080 con registri `A B C D E H L`, `PC`, `SP` e flag `C Z S P AC`.
- Memoria piatta da 64 KB e bus memoria mappato con protezione ROM.
- I/O separato a 256 porte, callback, terminale ASCII convenzionale su porte
  input `0` e output `1`.
- Decoder tabellare da 256 opcode; gli opcode non assegnati restituiscono
  `ErrUnimplementedOpcode`.
- Istruzioni v0.1: data movement, ALU, rotate, stack, salti/call/return,
  `RST`, `IN/OUT`, `EI/DI`, `HLT`.
- Timing aggregato per istruzione/ciclo macchina, READY/WAIT e interrupt esterno
  tramite jam instruction.
- Profili conservativi: `generic`, `altair-8800`, `imsai-8080`, `cpm-dev`.
- Debugger con trace JSON, breakpoint PC/opcode, watchpoint memoria e breakpoint
  I/O.

## Struttura

```text
retronet-8080/
|-- cmd/retronet-8080/   CLI
|-- cpu/                 core 8080 importabile
|-- machine/             profili, bus, terminale, front panel, debugger
|-- conformance/         suite sintetica senza ROM storiche
|-- docs/                documentazione italiana
|-- examples/            esempi raw e futuri programmi
`-- testdata/            vettori/versionati non storici
```

## Dipendenza hardware

Il core delega ALU e flag al bridge `github.com/retronet-labs/retronet-hardware/bridge/i8080`,
pubblicato da `retronet-hardware v0.5.0`.

```bash
docker build -t retronet/8080 .
```

## Limiti

- Nessuna ROM storica inclusa.
- Profili storici senza mappe S-100 o periferiche inventate.
- Timing aggregato, non transizioni elettriche pin-by-pin.
- Conformance sintetica interna, non ancora differenziale contro un secondo
  emulatore indipendente.
