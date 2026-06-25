# Esempi

Esempio raw minimo:

```text
3E 2A 76
```

I byte rappresentano:

```asm
MVI A, 0x2A
HLT
```

Su PowerShell:

```powershell
[IO.File]::WriteAllBytes("$env:TEMP\load-a.bin", [byte[]](0x3E, 0x2A, 0x76))
go run ./cmd/retronet-8080 -bin "$env:TEMP\load-a.bin" -trace -steps 8
```

Echo I/O con terminale convenzionale input `0`, output `1`:

```text
DB 00 D3 01 76
```

```powershell
[IO.File]::WriteAllBytes("$env:TEMP\echo.bin", [byte[]](0xDB, 0x00, 0xD3, 0x01, 0x76))
go run ./cmd/retronet-8080 -bin "$env:TEMP\echo.bin" -terminal-input Z -steps 8
```

Esempi assembly versionati arriveranno con il backend `i8080` di `retronet-asm`.
