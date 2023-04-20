// Package crawling implements helper functions around multiaddresses and peer
// addresses.
package crawling

import (
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	log "github.com/sirupsen/logrus"
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

// addrInfosToIDs extracts peer IDs from addresses.
func addrInfosToIDs(addrs []peer.AddrInfo) []peer.ID {
	peers := make([]peer.ID, len(addrs))
	for i, addr := range addrs {
		peers[i] = addr.ID
	}
	return peers
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
func stripLocalAddrs(pinfo peer.AddrInfo) peer.AddrInfo {
	// We skip local and private addresses and return a new peer.AddrInfo.
	// However, we create new MultiAddr objects to be on the safe side.

	strippedPinfo := peer.AddrInfo{
		ID:    pinfo.ID,
		Addrs: make([]ma.Multiaddr, 0),
	}
	for _, maddr := range pinfo.Addrs {
		if manet.IsPrivateAddr(maddr) || manet.IsIPLoopback(maddr) {
			continue
		}
		newAddr, err := ma.NewMultiaddr(maddr.String())
		if err != nil {
			log.WithField("err", err).Warn("Error creating multiaddr")
			continue
		}
		strippedPinfo.Addrs = append(strippedPinfo.Addrs, newAddr)
	}
	return strippedPinfo
}
