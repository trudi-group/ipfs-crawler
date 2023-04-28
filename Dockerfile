FROM golang:1.20-bullseye AS builder

WORKDIR /usr/src/ipfs-crawler/
# Download all dependencies first, this should be cached.
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make build

FROM debian:bullseye-slim AS runner

# Create a system user to drop into.
RUN groupadd -r ipfs \
  && useradd --no-log-init -r -g ipfs ipfs \
  && mkdir -p ipfs

# Enter our working directory.
WORKDIR libp2p-crawler

# Copy compiled binaries from builder.
COPY --from=builder /usr/src/ipfs-crawler/cmd/ipfs-crawler/ipfs-crawler ./libp2p-crawler
COPY --from=builder /usr/src/ipfs-crawler/dist/docker_entrypoint.sh .
COPY --from=builder /usr/src/ipfs-crawler/dist/config_ipfs.yaml ./config/config_ipfs.yaml
COPY --from=builder /usr/src/ipfs-crawler/dist/config_filecoin_mainnet.yaml ./config/config_filecoin_mainnet.yaml

# Set ownership.
RUN chown -R ipfs:ipfs ./libp2p-crawler

# Drop root.
USER ipfs

# Run the binary.
ENTRYPOINT ["./docker_entrypoint.sh","--config","./config/config_ipfs.yaml"]

