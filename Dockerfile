FROM alpine:latest AS build
WORKDIR /go/build
RUN apk update
RUN apk upgrade
RUN apk add --update go gcc g++
ADD . .
#RUN make
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o crawler cmd/ipfs-crawler/main.go

FROM alpine:latest
ENV LIBP2P_SWARM_FD_LIMIT=10000
ENV LIBP2P_ALLOW_WEAK_RSA_KEYS=true
WORKDIR /usr/local/bin
COPY --from=build /go/build/crawler ipfs_crawler
WORKDIR /ipfs
