# Etapa de build
FROM golang:1.21 AS builder

WORKDIR /app
COPY ./go_service .
RUN go build -o rinha .

CMD ["ls"]

# Etapa de execução
FROM debian:bookworm-slim AS runner

COPY --from=builder /app/rinha /rinha

CMD ["/rinha"]
