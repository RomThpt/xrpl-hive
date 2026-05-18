ARG baseimage=goxrpl
ARG tag=latest

FROM golang:1.24-alpine AS builder

ARG github=https://github.com/LeJamon/goXRPLd.git
ARG tag=main

RUN apk add --no-cache git gcc musl-dev pkgconf openssl-dev openssl-libs-static

WORKDIR /build
RUN git clone --depth 1 --branch $tag $github . || git clone $github . && git checkout $tag
RUN go build -o /goxrpl ./cmd/xrpld

FROM alpine:latest

RUN apk add --no-cache python3 bash ca-certificates

COPY --from=builder /goxrpl /usr/local/bin/goxrpl

# Write version.
RUN /usr/local/bin/goxrpl version > /version.txt 2>&1 || echo "goxrpl-unknown" > /version.txt

RUN mkdir -p /etc/goxrpl /tmp/goxrpl/db /hive-bin

ADD xrpl_start.sh /xrpl_start.sh
RUN chmod +x /xrpl_start.sh

EXPOSE 5005 6006 51235

ENTRYPOINT ["/xrpl_start.sh"]
