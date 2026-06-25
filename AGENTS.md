# Contesto operativo per agenti

## Obiettivo

Implementare e mantenere `retronet-8080`: emulatore Intel 8080 didattico,
testato, importabile e coerente con `go-4004` e `retronet-8008`.

## Architettura

- `cpu/`: core indipendente. Non importa `machine`.
- `machine/`: profili, bus memoria, I/O callback, terminale, front panel e debugger.
- `conformance/`: programmi sintetici senza ROM storiche.
- `cmd/retronet-8080/`: runner CLI.
- `docs/`: documentazione italiana.

## Decisioni da preservare

- Spazio memoria 8080 completo da 64 KB (`0x0000-0xFFFF`).
- Reset deterministico: CPU non halted, interrupt disabilitati, `PC=0`, `SP=0`.
- Stack in memoria, little-endian: `PUSH/CALL/RST` decrementano `SP`.
- Il byte PSW ha bit 1 sempre a 1; `PUSH/POP PSW` usano `A` + flag.
- Opcode non assegnati: `08 10 18 20 28 30 38 CB D9 DD ED FD`.
- I/O separato a 256 porte per input e output.
- Terminale convenzionale su input `0` e output `1`, non mappa storica.
- Nessuna ROM storica senza provenienza e licenza documentate.
- Profili storici conservativi: non inventare mappe Altair/IMSAI.
- Il core delega ALU e flag al bridge `retronet-hardware/bridge/i8080`.

## Verifica

```powershell
$env:GOCACHE='C:\work\source\retronet-8080\.gocache'
go test -count=1 ./...
go run ./cmd/retronet-8080 -conformance
go vet ./...
```

Prima di chiudere modifiche Go: `gofmt` e, se disponibile, `git diff --check`.
