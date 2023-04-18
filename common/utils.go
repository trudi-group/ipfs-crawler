package common

import (
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Parses an IPFS peer string and converts it into an ID + multiaddr. This is very useful when connecting to bootstrap nodes.
func ParsePeerString(text string) (*peer.AddrInfo, error) {
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
