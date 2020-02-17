package main

import (
	"fmt"
	"github.com/minio/sha256-simd"
	// kb "github.com/libp2p/go-libp2p-kbucket"
	"encoding/hex"
	"encoding/binary"
	"os"
	"sort"
	"math"
)

const (
	targetCPL = 24
	outFile = "preimages.csv"
)

func main() {
	numCombinations := int(math.Pow(2, targetCPL))
	numBytes := int(math.Floor(targetCPL/8))

	fmt.Printf("Generating %d hashes (targetCPL=%d). Outputting to %s\n", numCombinations, targetCPL, outFile)
	preImages := make(map[string]string, numCombinations)
	var unhashed = make([]byte, 32)
	
	// Keep track of keys (= hashes) so we can nicely sort them lateron
	var keys []string

	var i uint64
	for len(preImages) < numCombinations {
		thash := sha256.Sum256(unhashed)
		// fmt.Printf("%08b\n", thash[0])
		var s string
		for j := 0;j < numBytes;j++ {
			s += fmt.Sprintf("%08b", thash[j])
		}
		// s := fmt.Sprintf("%08b%08b", thash[0], thash[1])
		if _, ok := preImages[s]; ok {
			// item is present, do nothing
		} else {
			preImages[s] = hex.EncodeToString(unhashed)
			keys = append(keys, s)
		}
		// fmt.Println(unhashed[0])
		i++
		binary.LittleEndian.PutUint64(unhashed, i)
		if i % 10000 == 0 {
			fmt.Printf("i: %d. Len: %d\n", i, len(preImages))
		}
	}

	// write out the map
	// First, sort the keys
	sort.Strings(keys)
	file, err := os.OpenFile(outFile, os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(file, "hash;preimage\n")
	for _, k := range keys {
		fmt.Fprintf(file, "%s;%s\n", k, preImages[k])
	}
	file.Close()

}