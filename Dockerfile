FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGET=api
RUN go build -o app ./cmd/${TARGET}

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/app .
CMD ["./app"]
