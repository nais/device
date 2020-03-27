FROM golang:1.13-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN make test
RUN make alpine

FROM alpine:3.11
RUN apk add --no-cache ca-certificates git curl
WORKDIR /app
COPY --from=builder /src/bin/apiserver /app/apiserver
CMD ["/app/apiserver"]
