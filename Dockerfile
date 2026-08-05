FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o zili-api .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/zili-api .
COPY static ./static

EXPOSE 8081

CMD ["./zili-api"]

