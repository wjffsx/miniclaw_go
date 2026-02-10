FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o miniclaw_go cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/miniclaw_go .
COPY --from=builder /app/configs ./configs

RUN mkdir -p /app/data /app/sessions /app/memory /app/models

EXPOSE 18789

ENTRYPOINT ["./miniclaw_go"]