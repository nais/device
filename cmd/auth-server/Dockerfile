FROM golang:1.25-alpine AS builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . /workspace

RUN CGO_ENABLED=0 go build -o auth-server ./cmd/auth-server

FROM scratch
WORKDIR /

COPY --from=builder /workspace/auth-server .

ENTRYPOINT ["/auth-server"]
