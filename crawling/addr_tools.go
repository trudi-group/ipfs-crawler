package crawling
// static functions concerning the multiaddresses or peerAddrs
import (
	ma "github.com/multiformats/go-multiaddr"
	peer "github.com/libp2p/go-libp2p-core/peer"
	manet "github.com/multiformats/go-multiaddr-net"
	log "github.com/sirupsen/logrus"
)

func AddrInfoToID(addrs []*peer.AddrInfo) []peer.ID {
	peers := make([]peer.ID, len(addrs))
	for i, addr := range addrs {
		peers[i] = addr.ID
	}
	return peers
}

// Checks whether there are some addresses contained in array "new" that are not contained in array "old".
// Not the most sophisticated algorithm ever, but these arrays are never big, so no need for optimization here.
func FindNewMA(old []ma.Multiaddr, new []ma.Multiaddr) []ma.Multiaddr {
	var newAddrs []ma.Multiaddr
	var found bool
	for _, newaddr := range(new) {
		found = false
		for _, oldaddr := range(old) {
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
// stripLocalAddrs removes local adresses from an multiadress.
// Useful because a lot of the responses contain local adresses, which we cannot dial.
// :param pinfo: MultiAddr
// :return: new multiaddr with only non-public addresses
func stripLocalAddrs(pinfo peer.AddrInfo) peer.AddrInfo {
	// We skip local and private addresses and return a new peer.AddrInfo.
	// However, we create new MultiAddr objects to be on the safe side.

	strippedPinfo := peer.AddrInfo{
		ID: pinfo.ID,
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
