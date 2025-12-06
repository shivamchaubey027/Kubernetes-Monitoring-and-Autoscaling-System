# Dockerfile
FROM golang:1.23-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o watcherbot .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/watcherbot .
EXPOSE 8088
CMD ["./watcherbot"]
