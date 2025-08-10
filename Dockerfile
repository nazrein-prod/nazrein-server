
FROM golang:1.23-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY internal/ internal/
COPY migrations/ migrations/

RUN CGO_ENABLED=0 GOOS=linux go build -o nazrein

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=builder /build/nazrein .

COPY --from=builder /build/migrations ./migrations

USER nonroot:nonroot

CMD ["/app/nazrein"]
