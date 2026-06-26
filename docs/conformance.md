# Conformance Sintetica

La suite `conformance` esegue programmi piccoli e isolati contro il core e la
macchina generica:

```bash
go run ./cmd/retronet-8080 -conformance
```

Copre caricamenti, ALU, memoria indiretta, stack, salti condizionati, rotate,
I/O, `RST`, interrupt e READY/WAIT.

## Diagnostiche CP/M reali (8080EXM e affini)

Oltre alla suite sintetica, il core è validato contro le diagnostiche storiche
CP/M, eseguite localmente dal test `TestCPMDiagnosticROM`
([conformance/cpmdiag_test.go](../conformance/cpmdiag_test.go)). Le ROM **non
sono versionate** (copyright): vanno fornite a parte. Il test si salta se non ne
trova nessuna, così la suite resta verde per chi non le ha.

Esito misurato:

| ROM | Copertura | Esito |
| --- | --- | --- |
| TST8080.COM | CPU di base | CPU IS OPERATIONAL |
| 8080PRE.COM | preliminare ai flag | tests complete |
| **8080EXM.COM** | **exerciser esaustivo (tutte le istruzioni, tutti i flag via CRC)** | **tutti i gruppi PASS** |

Il superamento di **8080EXM** — il riferimento d'oro del settore — certifica
che la CPU è corretta bit-per-bit, **eseguendo l'aritmetica, i flag e le
rotazioni sull'ALU a porte logiche** di retronet-logic (tramite il bridge
`i8080`).

### Come eseguirle

Procurati il `.com` e indicalo con la variabile d'ambiente, oppure mettilo in
`conformance/testdata/diag/` (cartella *gitignored*). 8080EXM va lanciata in
modo esplicito (è troppo lenta per l'auto-discovery).

```powershell
# Windows PowerShell
$env:RETRONET_8080_DIAG_ROM = "C:\percorso\8080EXM.COM"
$env:RETRONET_8080_DIAG_MAXSTEPS = "0"   # nessun limite di step
go test ./conformance/ -run TestCPMDiagnosticROM -v -timeout 0
```

```bash
# bash
RETRONET_8080_DIAG_ROM=/percorso/8080EXM.COM RETRONET_8080_DIAG_MAXSTEPS=0 \
  go test ./conformance/ -run TestCPMDiagnosticROM -v -timeout 0
```

### Backend ALU: porte vs operatori Go

Il core usa per default l'ALU a **porte logiche** (`cpu.Gate`). 8080EXM su
porte richiede alcuni minuti. Impostando `RETRONET_8080_ALU=native` la stessa
diagnostica gira sul backend con **operatori Go** (`cpu.Native`) in pochi
secondi — comodo per la CI. I due backend sono garantiti **identici** (risultato
e ogni flag) dal test differenziale `TestGateVsNativeALUDifferential`
([cpu/alu_diff_test.go](../cpu/alu_diff_test.go)), quindi il risultato di 8080EXM
vale per entrambi.

```powershell
$env:RETRONET_8080_ALU = "native"   # poi rilancia il comando sopra: EXM in pochi secondi
```
