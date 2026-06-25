# Conformance Sintetica

La suite `conformance` esegue programmi piccoli e isolati contro il core e la
macchina generica:

```bash
go run ./cmd/retronet-8080 -conformance
```

Copre caricamenti, ALU, memoria indiretta, stack, salti condizionati, rotate,
I/O, `RST`, interrupt e READY/WAIT. Non sostituisce un confronto differenziale
con un secondo emulatore indipendente.
