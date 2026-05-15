ARG tag=latest

FROM rust:1.85-bookworm AS builder

ARG github=https://github.com/RomThpt/rxrpl.git
ARG tag=main

RUN apt-get update && apt-get install -y git protobuf-compiler clang libclang-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /build
RUN git clone --depth 1 --branch $tag $github . || git clone $github . && git checkout $tag
RUN cargo build --release --bin rxrpl

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y python3 bash ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/target/release/rxrpl /usr/local/bin/rxrpl

# Write version.
RUN /usr/local/bin/rxrpl --version > /version.txt 2>&1 || echo "rxrpl-unknown" > /version.txt

RUN mkdir -p /etc/rxrpl /var/lib/rxrpl/data /hive-bin

ADD xrpl_start.sh /xrpl_start.sh
RUN chmod +x /xrpl_start.sh

EXPOSE 5005 6006 51235

ENTRYPOINT ["/xrpl_start.sh"]
