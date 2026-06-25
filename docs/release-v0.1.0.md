# Checklist release v0.1.0

- [ ] `gofmt -l .` non produce output.
- [ ] `go vet ./...` termina senza errori.
- [ ] `go test -count=1 ./...` termina senza errori.
- [ ] `go run ./cmd/retronet-8080 -conformance` termina con tutti i casi verdi.
- [ ] README, docs e limiti noti descrivono lo stesso perimetro.
- [ ] `retronet-hardware` con bridge `i8080` e' disponibile come release taggata.

Limiti dichiarati:

- nessuna ROM storica distribuita
- niente CP/M in questo repo
- profili storici conservativi
- timing aggregato, non pin-level
