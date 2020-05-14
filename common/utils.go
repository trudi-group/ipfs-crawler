package common

import (
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"strings"
	"fmt"
	"os"

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

// Asks the user to enter "y" or "n". Also recognizes "yes" and "no" in all capitalizations.
// Anything unrecognized will is equivalent to "n".
func AskYesNo() bool {
	var response string
	positiveResp := []string{"y", "yes"}
	// negativeResp := []string{"n", "no"}

	_, err := fmt.Scanln(&response)
	if err != nil {
		return false
	}

	response = strings.ToLower(response)
	if containsOneResponse(response, positiveResp) {
		return true
	} else {
		return false
	}
}

func containsOneResponse(inputString string, resp []string) bool {
	for _, r := range resp {
		if strings.Contains(inputString, r) {
			return true
		}
	}
	return false
}

func CreateDirIfNotExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
		return nil
	}
	return err
}