# Architettura Intel 8080

Il package `cpu` modella l'Intel 8080 a livello di istruzione:

- registri `A`, `B`, `C`, `D`, `E`, `H`, `L`
- flag `Carry`, `Zero`, `Sign`, `Parity`, `AuxiliaryCarry`
- program counter e stack pointer a 16 bit
- memoria diretta da 64 KB
- I/O separato a 256 porte
- decoder tabellare da 256 opcode
- timing aggregato per istruzione e ciclo macchina

Lo stack e' memoria normale: `CALL`, `RET`, `RST`, `PUSH` e `POP` leggono e
scrivono tramite il bus `cpu.Memory`. Il reset del progetto e' deterministico:
registri e flag a zero, CPU eseguibile, interrupt disabilitati.

Il package `machine` costruisce sopra il core: bus memoria con regioni RAM/ROM,
callback I/O, terminale, front panel, READY/WAIT, interrupt esterno e debugger.
Il core non importa mai `machine`.
