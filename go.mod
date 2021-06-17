module ipfs-crawler

go 1.16

require (
	github.com/DataDog/zstd v1.4.8
	github.com/ipfs/go-bitswap v0.3.4
	github.com/ipfs/go-cid v0.0.7
	github.com/libp2p/go-libp2p v0.14.2
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-kad-dht v0.12.1
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	github.com/libp2p/go-msgio v0.0.6
	github.com/minio/sha256-simd v1.0.0
	github.com/multiformats/go-multiaddr v0.3.2
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
)

replace github.com/libp2p/go-libp2p-kad-dht v0.12.1 => github.com/harlequix/go-libp2p-kad-dht v0.12.2
