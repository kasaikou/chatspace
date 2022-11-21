FROM golang:1.18 as builder

WORKDIR /opt

COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN mkdir -p /opt/bin/ && \
    go build -o /opt/bin/app .

FROM debian:latest as runner

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates
COPY --from=builder /opt/bin/app /opt/bin/

EXPOSE 8080
ENTRYPOINT [ "/opt/bin/app" ]
