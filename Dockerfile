FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o zili-api .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/zili-api .
COPY --from=builder /app/static ./static
COPY --from=builder /app/sql ./sql

EXPOSE 8080

CMD ["./zili-api"]

