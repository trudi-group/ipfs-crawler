// Package crawling implements helper functions around multiaddresses and peer
// addresses.
package crawling

import (
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// parsePeerString arses an IPFS peer string and converts it into an
// ID + multiaddr. This is very useful when connecting to bootstrap nodes.
func parsePeerString(text string) (*peer.AddrInfo, error) {
	// Multiaddr
	if strings.HasPrefix(text, "/") {
		maddr, err := ma.NewMultiaddr(text)
		if err != nil {
			return nil, err
		}
		ainfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return nil, err
		}
		return ainfo, nil
	}
	return nil, peer.ErrInvalidAddr
}

// filterOutOldAddresses filters the addresses given in new with the addresses
// old, returning new addresses.
func filterOutOldAddresses(old []ma.Multiaddr, new []ma.Multiaddr) []ma.Multiaddr {
	var newAddrs []ma.Multiaddr
	var found bool
	for _, newaddr := range new {
		found = false
		for _, oldaddr := range old {
			if newaddr.Equal(oldaddr) {
				// We already know that address -> next
				found = true
				break
			}
		}
		if !found {
			newAddrs = append(newAddrs, newaddr)
		}
	}
	return newAddrs
}

// stripLocalAddrs removes local addresses from the given set of addresses.
// Returns a copy of the slice.
func stripLocalAddrs(mas []ma.Multiaddr) []ma.Multiaddr {
	out := make([]ma.Multiaddr, 0, len(mas))

	for _, maddr := range mas {
		if manet.IsPrivateAddr(maddr) || manet.IsIPLoopback(maddr) {
			continue
		}
		out = append(out, maddr)
	}

	return out
}
