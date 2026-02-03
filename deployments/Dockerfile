FROM golang:1.25.4-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/app /app/app

ENV CONFIG_PATH=/etc/app/config.yml

EXPOSE 8080

ENTRYPOINT ["/app/app"]
