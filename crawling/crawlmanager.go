package crawling

import (
	utils "ipfs-crawler/common"
	"fmt"
	"context"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"time"
	"os"
	"bufio"
	"encoding/hex"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"strings"
	// "strconv"
	// "errors"
	// swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
    "encoding/json"
    "io/ioutil"


)

const (
	// The date format which is appended to our crawl files. But seriously Go, wtf is this??
	filenameTimeFormat = "02-01-06--15:04:05"
	// These options should be configurable through the command line one day
	outPath = "crawls/"
	preImagePath = "precomputed_hashes/preimages.csv"
	numPreImages = 16777216
	writeToFileFlag = false
	cannaryFile = "configs/cannary.txt"
	sanity = true
)


// CrawlManager assigns nodes to be crawled to the workers and filters duplicates from the responses.
type CrawlManager struct {
	cacheFile string
	queueSize int
	InputQueue chan peer.AddrInfo
	onlineQueue chan peer.AddrInfo
	workQueue chan peer.AddrInfo
	knowQueue chan *NodeKnows
	// We use this map not only to store whether we crawled a node but also to store a nodes multiaddress
	crawled map[peer.ID][]ma.Multiaddr
	knows map[peer.ID][]peer.ID
	online map[peer.ID]bool
	quitMsg chan bool
	Done chan bool
	workers []*Worker
	ctx context.Context
	startTime time.Time
	ph *PreImageHandler
	errorChan chan error
	errorMap map[string]int
}
// PreImageHandler contains the precalculated hashes.
type PreImageHandler struct {
	preImages map[string]string
}
// NewCrawlManager creates a new crawlManager and loads the precalculated hashes.
// :param queueSize: size of the various message channels
func NewCrawlManager(queueSize int, cacheFile string) *CrawlManager {
	cm := &CrawlManager{
		queueSize: queueSize,
		cacheFile: cacheFile,
		InputQueue: make(chan peer.AddrInfo, queueSize),
		workQueue: make(chan peer.AddrInfo, queueSize),
		onlineQueue: make(chan peer.AddrInfo, queueSize),
		knowQueue: make(chan *NodeKnows, queueSize),
		crawled: make(map[peer.ID][]ma.Multiaddr),
		online: make(map[peer.ID]bool),
		knows: make(map[peer.ID][]peer.ID),
		quitMsg: make(chan bool),
		Done: make(chan bool),
		startTime: time.Now(),
		errorChan: make(chan error, queueSize),
		errorMap: make(map[string]int),
	}
	ctx, _ := context.WithCancel(context.Background())
	cm.ctx = ctx
	log.Info("Loading pre-images...")
	// log.WithField("numberOfPreImages", numPreImages).Info("Loading pre-images...")
	preImages, err := LoadPreimages(preImagePath, numPreImages)
	if err != nil {
		log.WithField("err", err).Error("Could not load pre-images. Continue anyway? (y/n)")
		if !utils.AskYesNo() {
			os.Exit(0)
		}
	}
	log.WithField("numPreImages", len(preImages)).Info("Successfully loaded pre-images.")

	ph := &PreImageHandler{
		preImages: preImages,
	}
	cm.ph = ph
	return cm
}
// CreateAndAddWorker creates a new worker for the crawlManager.
func (cm *CrawlManager) CreateAndAddWorker() {
	w := NewWorker(cm, len(cm.workers), cm.ctx)
	cm.AddWorker(w)
}
// AddWorker assings an existing worker to a manager.
func (cm *CrawlManager) AddWorker(w *Worker) {
	cm.workers = append(cm.workers, w)
}
// Stop initiates the shutdown of the crawling process.
func (cm *CrawlManager) Stop() {
	cm.quitMsg<-true
}

func (cm *CrawlManager) shutdownAndOutput() {
	// Perform sanity checks on output
	if sanity == true {
		checkCannaries(cm.knows)
        cm.saveOnlineNodes(cm.online, cm.crawled)
	}
	// Signal stop to workers first
	for _, w := range cm.workers {
		w.Stop()
	}
	cm.OutputVisitedPeers(writeToFileFlag)

}

// Run starts the crawling process.
func (cm *CrawlManager) Run() {
	// Start workers first
	for _, w := range cm.workers {
		go w.Run()
	}

	// We stop the crawl whenever the queues are empty and nothing has happened for 2 minutes
	for {
		log.WithFields(log.Fields{
			"inputQueueLength": len(cm.InputQueue),
			"workQueueLength": len(cm.workQueue),
		}).Info("CrawlManager: New loop run.")
		idleTimer := time.NewTimer(1*time.Minute)
		select {
			case node := <-cm.InputQueue:
				// First, stop the idle timer. The following code is from the docs, apparently there are race conditions
				// with Stop() and the timer channel we're reading from.
				if !idleTimer.Stop() {
					<-idleTimer.C
				}
				// Check if we've already seen that node. If it was online the last time, we do not crawl it again.
				// If we learned new addresses of that node, we try to crawl it again.
				oldAddrs, crawled := cm.crawled[node.ID]
				_, online := cm.online[node.ID]
				if crawled && online {
					continue
				}
				if crawled && !online {
					// Check if there are any new addresses. If so, connect to them
					newAddrs := FindNewMA(oldAddrs, stripLocalAddrs(node).Addrs)
					if len(newAddrs) == 0 {
						// Nothing new, don't bother dialing again
						continue
					}
					cm.crawled[node.ID] = append(cm.crawled[node.ID], newAddrs...)
					cm.workQueue<-peer.AddrInfo{
						ID: node.ID,
						Addrs: newAddrs,
					}
					continue
				}
				// If not, we remember that we've seen it and add it to the work queue, so that a worker will eventually crawl it.
				cm.crawled[node.ID] = node.Addrs
				cm.workQueue<-node

				// Just out of curiousity, if the Queue is starting to overflow
				if len(cm.workQueue) > 40000 {
					log.WithField("workQueueLength", len(cm.workQueue)).Warning("WorkQueue getting long.")
				}
        if len(cm.InputQueue) > 40000 {
					log.WithField("inputQueueLength", len(cm.InputQueue)).Warning("inputQueue getting long.")
				}
			case node := <-cm.onlineQueue:
				// A peer that we saw was online, store that
				// First, check if we've seen the peer. This should actually never fail:
				_, ok := cm.crawled[node.ID]
				if !ok {
					log.WithField("nodeID", node.ID).Panic("Crawled node but not stored in seen-map.")
				}
				cm.online[node.ID] = true

			case k := <-cm.knowQueue:
				// Store that knowledge for later when outputting the peers
				cm.knows[k.id] = AddrInfoToID(k.knows)

			case <-idleTimer.C:
				// Stop the crawl
				log.Info("Idle timer fired, stopping the crawl.")
				// If the queues are not empty, something went wrong
				if len(cm.InputQueue) != 0 || len(cm.workQueue) != 0 {
					log.WithFields(log.Fields{
						"inputQueueLength": len(cm.InputQueue),
						"workQueueLength": len(cm.workQueue),
					}).Panic("Queues are not empty!")
				}
				go cm.Stop()

			case err:=<-cm.errorChan:
				cm.errorMap[fmt.Sprintf("%T", err)] += 1

			case <-cm.quitMsg:
				log.Debug("CrawlManager quitting....")
				cm.shutdownAndOutput()
				cm.Done<-true
				return
		}
	}
}
// OutputVisitedPeers writes the learned information about IPFS to a file.
// :param toFile: indicates whether to create a file
func (cm *CrawlManager) OutputVisitedPeers(toFile bool) {
	// Write to file...
	start := cm.startTime.Format(filenameTimeFormat)
	end := time.Now().Format(filenameTimeFormat)
	if toFile {
		vf, err := os.OpenFile(outPath + cm.generateFilename("visitedPeers", ".csv", start, end), os.O_CREATE | os.O_RDWR, 0666)
		if os.IsNotExist(err) {
			os.Mkdir(outPath, 0666)
		} else if err != nil {
			panic(err)
		}

		// cm.crawled is a map from node.ID -> []multiaddr
		for node, addrs := range cm.crawled {
			// This works because Boolean zero-value is "false"
			// cm.online is a map from node.ID -> bool
			_, exists := cm.online[node]
			fmt.Fprintf(vf, "%s;%s;%t\n", node, addrs, exists)
			// fmt.Printf("%s;%t\n", node, online)
		}
		vf.Sync()
		vf.Close()

		f, err := os.OpenFile(outPath + cm.generateFilename("peerGraph", ".csv", start, end), os.O_CREATE | os.O_RDWR, 0666)
		fmt.Fprintf(f, "SOURCE;TARGET;ONLINE\n")
		for node, knowsNodes := range cm.knows {
			for _, val := range knowsNodes {
				on, _ := cm.online[val]
				fmt.Fprintf(f, "%s;%s;%t\n", node, val, on)
			}
		}
		f.Sync()
		f.Close()

	// ... or just output to terminal
	} else {
		// for node, _ := range cm.crawled {
		// 	// Again, zero-value is "false"
		// 	val, exists := cm.online[node]
		// 	fmt.Printf("%s;%s;%t\n", node, val, exists)
		// }
		// fmt.Println("============")
		// for node, knowsNodes := range cm.knows {
		// 	for _, val := range knowsNodes {
		// 		on, _ := cm.online[val]
		// 		fmt.Printf("%s;%s;%t\n", node, val, on)
		// 	}
		// }
	}
	fmt.Println("######## ErrorMap ###########")
	fmt.Println(cm.errorMap)
	fmt.Println("#############################")
}

func (cm *CrawlManager) generateFilename(prefix, suffix, start, end string) string {
	return prefix + "_" + start + "_" + end + ".csv"
}

func AddrInfoToID(addrs []*peer.AddrInfo) []peer.ID {
	peers := make([]peer.ID, len(addrs))
	for i, addr := range addrs {
		peers[i] = addr.ID
	}
	return peers
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
	unhashed, err := hex.DecodeString(ph.preImages[s])
	if err != nil {
		panic(err)
	}
	return unhashed
}

// This function reads the pre-computed preimages from a file and puts it into a map.
// We can also determine the number of lines by just reading it, but it's not worth the hassle if this stays basically constant.
func LoadPreimages(path string, mapsize int) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	preImages := make(map[string]string, mapsize)

	scanner := bufio.NewScanner(file)
	// Throw away the header line
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, ";")
		preImages[splitLine[0]] = splitLine[1]
	}

	return preImages, nil
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
// checkCannaries is a sanity check which tests whether certain nodes have been seen during the crawling.
func checkCannaries (onlineMap map[peer.ID][]peer.ID)  {
	file, err := os.Open(cannaryFile)
	if err != nil {
		log.WithField("err", err).Error("cannot open cannary file")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cannary_text := scanner.Text()
		// Ignore lines that are commented out
		if strings.HasPrefix(cannary_text, "//") {
			continue
		}
		fmt.Println(cannary_text)
		cannary, err := utils.ParsePeerString(cannary_text)
		if err != nil {
			log.WithField("err", err).Error("Error parsing cannary.")
			continue
		}
		online, exists := onlineMap[cannary.ID]
		if !exists {
			log.WithFields(log.Fields{
			"cannary": cannary,
			}).Error("Sanity: Cannary not found.")
		} else if online == nil {
			log.WithFields(log.Fields{
				"cannary": cannary,
				}).Error("Sanity: Cannary found.")
		}
	}
}
// saveOnlineNodes saves nodes for caching purposes.
func (cm *CrawlManager) saveOnlineNodes(online map[peer.ID]bool, crawled map[peer.ID][]ma.Multiaddr)  {
    nodesSave := []peer.AddrInfo{}
    // BUG this saves every node, not only online nodes
    for id, _ :=  range online {
        if true {
            recreated := peer.AddrInfo{
                ID: id,
                Addrs: crawled[id],
            }
            nodesSave = append(nodesSave, recreated)
        }
    }
    marshalled, _ := json.Marshal(nodesSave)
    err := ioutil.WriteFile(cm.cacheFile, marshalled, 0644)
    if err != nil {
        log.WithField("err", err).Error("Error writting to cacheFile.")
        return
    }

}
// RestoreNodeCache restores a previously cached file of nodes.
func RestoreNodeCache(path string) ([]*peer.AddrInfo, error)  {
    nodedata, err := ioutil.ReadFile(path)
    if err != nil {
        log.WithField("err", err).Error("Error reading from cacheFile.")
        return nil, err
    }
    var result []peer.AddrInfo
    err = json.Unmarshal(nodedata, &result)
    if err != nil {
        log.WithField("err", err).Error("Error unmarshalling from cacheFile.")
        return nil, err
    }
    var out []*peer.AddrInfo
    // switch to pointers to fullfil requirements of main.go... because this is stupid
    for _, val := range result{
        out = append(out, &val)
    }
    return out, nil
}

func (cm *CrawlManager) GetInputQueueLen() int {
	return len(cm.InputQueue)
}

func (cm *CrawlManager) GetWorkQueueLen() int {
	return len(cm.workQueue)
}

func (cm *CrawlManager) GetQueueSize() int {
	return cm.queueSize
}