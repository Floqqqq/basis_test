FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./

COPY . .
ARG TARGET=api
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o app ./cmd/${TARGET}

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/app .
CMD ["./app"]
