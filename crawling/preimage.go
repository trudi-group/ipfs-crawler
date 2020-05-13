package crawling
import(
  "bufio"
  "strings"
  "os"
  "fmt"
  "encoding/hex"
  peer "github.com/libp2p/go-libp2p-core/peer"
  kb "github.com/libp2p/go-libp2p-kbucket"
  "github.com/DataDog/zstd"
)


type PreImageHandler struct {
	PreImages map[string]string
}

func LoadPreimages(path string, mapsize int) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	preImages := make(map[string]string, mapsize)
    var scanner *bufio.Scanner
    if strings.HasSuffix(path, ".zst") {
        compressed := zstd.NewReader(file)
        scanner = bufio.NewScanner(compressed)
    }else{
        scanner = bufio.NewScanner(file)
    }

	// Throw away the header line
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, ";")
		preImages[splitLine[0]] = splitLine[1]
	}

	return preImages, nil
	// file, err := os.Open(path)
	// if err != nil {
	// 	return nil, err
	// }
	// defer file.Close()
	// preImages := make(map[string]string, mapsize)

	// scanner := bufio.NewScanner(file)
	// // Throw away the header line
	// scanner.Scan()
	// for scanner.Scan() {
	// 	line := scanner.Text()
 //    fmt.Println(line)
	// 	splitLine := strings.Split(line, ";")
	// 	preImages[splitLine[0]] = splitLine[1]
	// }

	// return preImages, nil
}

// Given a common prefix length and the ID of the peer we're asking, this function builds an approriate binary string with
// the target CPL and returns the corresponding pre-image.
func (ph *PreImageHandler) FindPreImageForCPL(targetPeer peer.AddrInfo, cpl uint8) []byte {
	// Roadmap:
	// * We take the target's ID until CPL -> we have a common prefix of at least this length
	// * We then flip the next bit of the ID so we're sure to be different
	// * Convert the resulting bytes to string and look up the preimage in our database

	// ToDo: this could be generic
	if cpl > 23 {
		panic("CPL > 23 not possible.")
	}

	// Since the CPL could span multiple bytes, we have to determine in which byte we work
	var byteNum uint8
	byteNum = cpl/8

	// As well as the position within the byte
	bitPosition := cpl%8

	// We cannot work with the multihash, so use the IPFS-internal function to convert the peerID multihash.
	// Practically this means just hashing
	binID := kb.ConvertPeerID(targetPeer.ID)

	// Until bitPosition-1 we want to take the target's ID. The bit at bitPosition should be inverted to the ID.
	// So we take that as well and build an approriate bitmask for this task
	var mask uint8
	for i := 0; uint8(i) <= bitPosition; i++ {
		mask = mask>>1
		mask += 0x80
	}
	maskedID := binID[byteNum] & mask

	// Now let's flip the last bit
	var xorMask uint8
	xorMask = 0x80>>(bitPosition)
	maskedID = maskedID ^ xorMask

	// Now we have to put the pieces together into a string that we can use in our map
	var s string
	for j := 0; uint8(j) < byteNum; j++ {
		s += fmt.Sprintf("%08b", binID[j])
	}
	s += fmt.Sprintf("%08b", maskedID)

	// ToDo: Related to above: this could be generic
	for j := 0; uint8(j) < 2 - byteNum; j++ {
		s += "00000000"
	}
	// Lookup the preimage in our "database"
	unhashed, err := hex.DecodeString(ph.PreImages[s])
	if err != nil {
		panic(err)
	}
	return unhashed
}
