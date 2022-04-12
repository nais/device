FROM golang:1.18-alpine as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . /workspace

# Build
RUN CGO_ENABLED=0 go build -o auth-server .

FROM alpine:3.15
WORKDIR /

COPY --from=builder /workspace/auth-server .
USER 65532:65532

ENTRYPOINT ["/auth-server"]