# README

## Compute the pre-images

The crawler relies on a collection of pre-images. To compute them, simply execute the compute.go file in precomputed_hashes/

## Libp2p complains about keylengths

Libp2p uses minimum keylenghts of [2048 bit](https://github.com/libp2p/go-libp2p-core/blob/master/crypto/rsa_common.go), whereas IPFS uses [512 bit](https://github.com/ipfs/infra/issues/378).
Therefore, the crawler can only connect to one IPFS bootstrap node and refuses a connection with the others, due to this key length mismatch.
The environment variable that is used to change the behavior of libp2p is, for some reason, read before the main function of the crawler is executed. So it should be started with, e.g.:

```export LIBP2P_ALLOW_WEAK_RSA_KEYS="" && go run cmd/ipfs-crawler/main.go```

## Bootstrap Peers

By default, the crawler uses the bootstrappeer list provided in ```bootstrappeers.txt```. The file is assumed to contain one multiaddress in each line.
Lines starting with a comment ```//``` will be ignored.
To get the default bootstrap peers of an IPFS node, simply run ```./ipfs bootstrap list > bootstrappeers.txt```.