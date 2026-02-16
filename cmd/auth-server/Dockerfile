FROM golang:1.26-alpine AS builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . /workspace

RUN CGO_ENABLED=0 go build -o auth-server ./cmd/auth-server

FROM scratch
WORKDIR /

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /workspace/auth-server .

ENTRYPOINT ["/auth-server"]
