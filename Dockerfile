# multi-stage build

FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
