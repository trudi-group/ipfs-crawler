#!/bin/bash -e

# Build image
docker build -t trudi-group/ipfs-crawler .

# Extract binary
docker create --name extract trudi-group/ipfs-crawler
mkdir -p out
docker cp extract:/libp2p-crawler/libp2p-crawler ./out/libp2p-crawler
docker rm extract

