// Package main implements the hash-precomputation binary to compute preimages
// for the libp2p Kademlia crawler.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/minio/sha256-simd"
)

const (
	// Targeted common prefix length in bits.
	targetCPL = 24
	outFile   = "preimages.csv"
)

func main() {
	numCombinations := int(math.Pow(2, targetCPL))
	numBytes := int(math.Floor(targetCPL / 8))

	fmt.Printf("Generating %d hashes (targetCPL=%d). Writing to %s\n", numCombinations, targetCPL, outFile)

	// Map of hashes (as binary strings) to their preimage.
	preimages := make(map[string]string, numCombinations)
	preimage := make([]byte, 32)

	// Keep track of keys (= hashes) so we can nicely sort them later
	var keys []string

	var i uint64
	for len(preimages) < numCombinations {
		binary.LittleEndian.PutUint64(preimage, i)
		hash := sha256.Sum256(preimage)

		var encodedHash string
		for j := 0; j < numBytes; j++ {
			encodedHash += fmt.Sprintf("%08b", hash[j])
		}

		if _, ok := preimages[encodedHash]; !ok {
			// New prefix, record it.
			preimages[encodedHash] = hex.EncodeToString(preimage)
			keys = append(keys, encodedHash)
		}

		i++
		if i%10000 == 0 {
			fmt.Printf("i: %d. Len: %d\n", i, len(preimages))
		}
	}

	// Sort the keys for writing
	sort.Strings(keys)

	// Write results
	file, err := os.OpenFile(outFile, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		panic(err)
	}
	_, err = fmt.Fprintf(file, "hash;preimage\n")
	if err != nil {
		panic(err)
	}

	for _, k := range keys {
		_, err = fmt.Fprintf(file, "%s;%s\n", k, preimages[k])
		if err != nil {
			panic(err)
		}
	}

	err = file.Close()
	if err != nil {
		panic(err)
	}
}
