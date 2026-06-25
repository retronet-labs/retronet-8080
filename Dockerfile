FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY . .

RUN go build -o /out/retronet-8080 ./cmd/retronet-8080

FROM alpine:latest

WORKDIR /app
COPY --from=builder /out/retronet-8080 /app/retronet-8080

ENTRYPOINT ["/app/retronet-8080"]
CMD ["-h"]
