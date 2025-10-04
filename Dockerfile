FROM golang:alpine AS builder
WORKDIR /build
COPY i.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o i i.go

FROM scratch
COPY --from=builder /build/i /i
ENTRYPOINT ["/i"]