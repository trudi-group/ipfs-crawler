FROM golang:1.23-bullseye AS builder

WORKDIR /usr/src/ipfs-crawler/
# Download all dependencies first, this should be cached.
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -v -o ipfs-crawler cmd/ipfs-crawler/main.go

FROM debian:bullseye-slim AS runner

# Enter our working directory.
WORKDIR libp2p-crawler

# Copy compiled binaries from builder.
COPY --from=builder /usr/src/ipfs-crawler/ipfs-crawler ./libp2p-crawler
COPY --from=builder /usr/src/ipfs-crawler/dist/docker_entrypoint.sh .
COPY --from=builder /usr/src/ipfs-crawler/dist/config_ipfs.yaml ./config/config_ipfs.yaml
COPY --from=builder /usr/src/ipfs-crawler/dist/config_filecoin_mainnet.yaml ./config/config_filecoin_mainnet.yaml

# Link IPFS config to be executed by default
RUN ln -s ./config/config_ipfs.yaml config.yaml

# Run the binary.
ENTRYPOINT ["./docker_entrypoint.sh", "--config", "config.yaml"]