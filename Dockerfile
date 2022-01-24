# Builder image
FROM golang:1.16 AS builder

WORKDIR /builder-tmp

# Populate module cache
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 go build -o /vm-spinner *.go

# Final image
FROM debian:11-slim

RUN apt update && apt install -y curl

RUN curl -sLO https://releases.hashicorp.com/vagrant/2.2.19/vagrant_2.2.19_x86_64.deb
RUN dpkg -i vagrant_2.2.19_x86_64.deb

COPY --from=builder /vm-spinner /vm-spinner

ENTRYPOINT [ "/vm-spinner" ]
