build:
	go build cmd/ipfs-crawler/main.go
	mv main cmd/ipfs-crawler/crawler
	echo "export LIBP2P_ALLOW_WEAK_RSA_KEYS="" && ./cmd/ipfs-crawler/crawler" > start_crawl
	chmod u+x start_crawl

preimages:
	go build cmd/hash-precomputation/main.go
	mv main cmd/hash-precomputation/compute_preimages
	./cmd/hash-precomputation/compute_preimages
	mkdir -p precomputed_hashes
	mv preimages.csv precomputed_hashes/preimages.csv

clean:
	rm cmd/ipfs-crawler/crawler
	rm start_crawl
# 	rm cmd/hash-precomputation/compute_preimages

all: preimages build