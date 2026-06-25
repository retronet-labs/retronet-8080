# CLI

Il comando principale e':

```bash
go run ./cmd/retronet-8080 -bin programma.bin -steps 1000
```

Flag utili:

- `-bin`: carica un binario raw.
- `-addr`: indirizzo di caricamento, default `0x0000`.
- `-pc`: PC iniziale, default uguale a `-addr`.
- `-steps`: limite di istruzioni.
- `-trace`: stampa il disassembly prima di ogni istruzione.
- `-disasm N`: disassembla N istruzioni senza eseguire.
- `-profiles`: elenca profili macchina.
- `-rom nome=file`: carica una ROM locale in uno slot di profilo.
- `-input porta=valore`: inizializza una porta input.
- `-terminal-input`: abilita il terminale ASCII convenzionale.
- `-trace-json`: scrive eventi strutturati JSON Lines.
- `-conformance`: esegue la suite sintetica integrata.

Il terminale usa input port `0` e output port `1` per convenzione emulativa.
