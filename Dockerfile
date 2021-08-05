FROM golang:1.16 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the source
COPY . .

# Build
RUN GOOS=linux GOARCH=amd64 make build

FROM debian:buster-slim

RUN apt-get update && apt-get install -y iptables && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --set iptables /usr/sbin/iptables-legacy

COPY --from=builder /workspace/bin/fwmark-exporter /fwmark-exporter

ENTRYPOINT ["/fwmark-exporter"]
