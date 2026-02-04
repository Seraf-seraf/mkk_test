FROM golang:1.25.4-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o /out/app ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/app /app/app
COPY api/openapi.yml /app/api/openapi.yml
COPY internal/migrations /app/internal/migrations
COPY configs/config.yml /app/configs/config.yml

EXPOSE 8080

ENTRYPOINT ["/app/app"]
