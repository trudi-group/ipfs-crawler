build:
	go build cmd/ipfs-crawler/main.go
	mv main cmd/ipfs-crawler/ipfs-crawler

preimages:
	go build cmd/hash-precomputation/main.go
	mv main cmd/hash-precomputation/hash-precomputation
	./cmd/hash-precomputation/hash-precomputation
	mkdir -p precomputed_hashes
	mv preimages.csv precomputed_hashes/preimages.csv

clean:
	rm cmd/ipfs-crawler/crawler

all: preimages build
