package main

import (
	crawlLib "ipfs-crawler/crawling"
	utils "ipfs-crawler/common"

	"fmt"
	"time"
	"os"
	"os/signal"
	"bufio"
	"strings"

	peer "github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
)

const (
	numWorkers = 1
	// connectTimeout = 2 * time.Second
	// Indicates whether nodes should be cached during crawl runs to speed up the next successive crawl
	useCache = true
	// File where the nodes between crawls are cached (if caching is enabled)
	cacheFile = "nodes.cache"
	// Time format of log entries. Go, why you so ugly?
	logTimeFormat = "15:04:05"
	// Log level. Debug contains a lot but is very spammy
	logLevel = log.InfoLevel
	// File which contains the bootstrap peers
	bootstrapFile = "configs/bootstrappeers.txt"
	// Buffersize of each queue that we are using for communication between threads
	queueSize = 64384

)

	// FOR TESTING PURPOSES: OUR LOCAL NODE
	// "/ip4/127.0.0.1/tcp/4003/ipfs/QmamSnfS9bVjGgJJ57hznpCyMnesAtD3BidU8gfFBwUD7U", // local node

// TODO:
// * More robust error handling when connecting or receiving messages
// * Are relays used when connecting?

func main() {
	// There's a clash between libp2p (2024) and ipfs (512) minimum key lenghts -> set it to the one used in IPFS.
	// Since libp2p ist initialized earlier than our main() function we have to set it via the command line.

	// Setting up the logging
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = logTimeFormat
	// formatter.DisableSorting = true
	// Don't truncate the levels
	formatter.DisableLevelTruncation = true
	log.SetFormatter(formatter)
	log.SetLevel(logLevel)


	// Let's go!
	log.Info("Thank you for running our IPFS Crawler!")

	// First, check whether the weak RSA keys environment variable is set
	_, weakKeysAllowed := os.LookupEnv("LIBP2P_ALLOW_WEAK_RSA_KEYS")
	log.WithField("weak_RSA_keys", weakKeysAllowed).Info("Checking whether weak RSA keys are allowed...")
	if !weakKeysAllowed {
		log.Error("Weak RSA keys are *disabled*. The crawl will most likely return garbage, since " +
		"it will not be able to connect to the majority of nodes. Do you really want to continue? (y/n)")
		if !utils.AskYesNo() {
			os.Exit(0)
		}
	}

	// Second, check if the pre-image file exists
	cm := crawlLib.NewCrawlManager(queueSize, cacheFile)
	log.WithField("numberOfWorkers", numWorkers).Info("Creating workers...")
	for i := 0; i < numWorkers; i++ {
		cm.CreateAndAddWorker()
	}

	go cm.Run()

	bootstrappeers, err := LoadBootstrapList(bootstrapFile)
	if err != nil {
		panic(err)
	}
    if useCache {
        cachedNodes, err := crawlLib.RestoreNodeCache(cacheFile)
        if err == nil{
        	log.WithField("amount", len(cachedNodes)).Info("Adding cached peer to crawl queue.")
            bootstrappeers = append(bootstrappeers, cachedNodes...)
        }
    }

	for _, p := range bootstrappeers {
		log.WithField("addr", p).Debug("Adding bootstrap peer to crawl queue.")
		// fmt.Printf("[%s] Adding bootstrap peer to crawl queue: %s\n", Timestamp(), ainfo)
		cm.InputQueue<- *p
	}

	// Catch strg+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
    	for sig := range c {
    		fmt.Println(sig)
        	ShutDown(cm)
    	}
	}()
	exit := <-cm.Done
	// Fuck you go compiler
	_ = exit
	log.WithFields(log.Fields{
		"inputQueueLength": cm.GetInputQueueLen(),
		"workQueueLength": cm.GetWorkQueueLen(),
	}).Info("Exit successful.")
	os.Exit(0)
}

func ShutDown(cm *crawlLib.CrawlManager) {
	cm.Stop()
	time.Sleep(10*time.Second)
	// fmt.Println("======================")
	// cm.OutputVisitedPeers(true)
	log.WithFields(log.Fields{
		"inputQueueLength": cm.GetInputQueueLen(),
		"workQueueLength": cm.GetWorkQueueLen(),
	}).Info("Exit successful.")
	os.Exit(0)
}

// Parses a file containing bootstrap peers. It assumes a text file with a multiaddress on each line.
// It will ignore lines starting with a comment "//"
func LoadBootstrapList(path string) ([]*peer.AddrInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file line by line and parse the multiaddress string
	var bootstrapMA []*peer.AddrInfo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Ignore lines that are commented out
		if strings.HasPrefix(line, "//") {
			continue
		}
		ainfo, err := utils.ParsePeerString(line)
		if err != nil {
			log.WithField("err", err).Error("Error parsing bootstrap peers.")
			return nil, err
		}
		bootstrapMA = append(bootstrapMA, ainfo)
	}

	return bootstrapMA, nil

}
