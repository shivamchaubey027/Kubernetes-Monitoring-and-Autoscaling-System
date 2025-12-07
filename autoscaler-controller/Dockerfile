# Stage 1: The Build Stage
FROM golang:1.24 AS builder 
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /autoscaler-controller ./main.go

# Stage 2: The Final (Scratch) Image
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /autoscaler-controller .

CMD ["./autoscaler-controller"]