package crawling

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/zstd"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
)

// MaxCPL is the maximum prefix length we can probe.
const MaxCPL = 24

// The PreimageHandler handles selection of the correct preimages to extract
// information from specific Kademlia buckets of a peer.
type PreimageHandler struct {
	// This stores preimages for Kademlia ID prefixes.
	// Each preimage is an 8-byte array, stored as big endian within a uint64.
	// The index is a uint32 of which the _lower_ three bytes denote the 24-bit
	// prefix. The upper byte must be zero.
	preimages [0x01 << MaxCPL]uint64
}

// LoadPreimages loads precomputed preimages from a potentially Zst-compressed
// file.
// Hashes in the file must be presented as binary strings, whereas preimages
// are hex-encoded 8-byte binary values.
func LoadPreimages(path string) (*PreimageHandler, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	preimages := [0x01 << MaxCPL]uint64{}
	var scanner *bufio.Scanner
	if strings.HasSuffix(path, ".zst") {
		compressed := zstd.NewReader(file)
		defer func() { _ = compressed.Close() }()
		scanner = bufio.NewScanner(compressed)
	} else {
		scanner = bufio.NewScanner(file)
	}

	// Throw away the header line
	scanner.Scan()
	// Decode input lines
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ";")

		// Extract the target prefix.
		var b1, b2, b3 uint8
		_, err := fmt.Sscanf(split[0], "%08b%08b%08b", &b1, &b2, &b3)
		if err != nil {
			return nil, fmt.Errorf("unable to decode target: %w", err)
		}
		target := uint32(b1)<<16 |
			uint32(b2)<<8 |
			uint32(b3)<<0

		// Extract the preimage.
		preimage, err := hex.DecodeString(split[1])
		if err != nil {
			return nil, fmt.Errorf("unable to decode preimage: %w", err)
		}
		if len(preimage) != 8 {
			return nil, fmt.Errorf("expected 8-byte preimage, got %d bytes", len(preimage))
		}

		// Store within a uint64.
		preimageUint := binary.BigEndian.Uint64(preimage)

		preimages[target] = preimageUint
	}

	return &PreimageHandler{preimages: preimages}, nil
}

// Given a common prefix length and the ID of the peer we're asking, this
// function builds an appropriate binary string with the target CPL and returns
// the corresponding pre-image.
func (ph *PreimageHandler) findPreImageForCPL(targetPeer peer.ID, cpl uint8) []byte {
	// Roadmap:
	// - Convert target peer ID to Kademlia keyspace
	// - Take the first three bytes
	// - Flip the bit at position cpl+1, i.e., make sure we have a common prefix
	//	 of length cpl, and the bit immediately after that is flipped.
	// - Convert the three bytes to a uint32 for lookup, return the preimage

	if cpl > MaxCPL-1 {
		panic(fmt.Sprintf("CPL > %d not calculated", MaxCPL-1))
	}

	// The peer ID is given as a multihash, which needs to be mapped onto the
	// Kademlia ID space first.
	// In practice, this means it's SHA256 hashed.
	binID := kb.ConvertPeerID(targetPeer)

	// Create uint32 from that, which we need for indexing.
	target := uint32(binID[0])<<24 |
		uint32(binID[1])<<16 |
		uint32(binID[2])<<8

	// Flip the bit immediately after the common prefix.
	target ^= uint32(0x80000000) >> cpl
	// Make sure we occupy the lower three bytes, for indexing.
	target = target >> 8

	// Lookup, convert to slice
	preimageUint := ph.preimages[target]
	preimage := make([]byte, 8)
	binary.BigEndian.PutUint64(preimage, preimageUint)

	log.Debugf("search for ID %08b%08b%08b, CPL=%02d, computed target %024b, returning %s", binID[0], binID[1], binID[2], cpl, target, hex.EncodeToString(preimage[:]))

	return preimage
}
