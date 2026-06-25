# Istruzioni

La v0.1 implementa le famiglie principali dell'Intel 8080:

- data movement: `MOV`, `MVI`, `LXI`, `LDA`, `STA`, `LHLD`, `SHLD`, `LDAX`,
  `STAX`, `XCHG`, `XTHL`, `SPHL`
- ALU e flag: `ADD`, `ADC`, `SUB`, `SBB`, `ANA`, `XRA`, `ORA`, `CMP`, immediati,
  `INR`, `DCR`, `INX`, `DCX`, `DAD`, `DAA`, `CMA`, `CMC`, `STC`
- rotate e controllo: `RLC`, `RRC`, `RAL`, `RAR`, `JMP`, `Jcc`, `CALL`, `Ccc`,
  `RET`, `Rcc`, `RST`, `PCHL`, `PUSH`, `POP`, `NOP`, `HLT`, `IN`, `OUT`,
  `EI`, `DI`

Opcode non assegnati:

```text
08 10 18 20 28 30 38 CB D9 DD ED FD
```

Questi byte restituiscono `ErrUnimplementedOpcode`, invece di essere trattati
come alias silenziosi.
