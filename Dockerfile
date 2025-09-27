
FROM golang:1.23-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o nazrein


FROM alpine:3.18 AS doppler
WORKDIR /doppler

# Download Doppler CLI binary
RUN apk add --no-cache curl \
    && curl -Ls https://cli.doppler.com/install.sh | sh


FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

COPY --from=builder /build/nazrein .
COPY --from=builder /build/migrations ./migrations
COPY --from=doppler /usr/local/bin/doppler /usr/local/bin/doppler

USER nonroot:nonroot

CMD ["doppler", "run", "--", "/app/nazrein"]
