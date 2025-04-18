FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify && go mod tidy
COPY . .
RUN go build -o gomcp && chmod +x ./gomcp 

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app .

EXPOSE 9090
EXPOSE 11434

ENTRYPOINT ["./gomcp"]
