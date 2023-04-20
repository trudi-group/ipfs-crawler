package crawling

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// A CrawlerWorker is a libp2p host which is used to crawl the network.
// It should support concurrent crawls.
// It should also execute any plugins on the discovered connectable nodes.
type CrawlerWorker interface {
	// CrawlPeer crawls the given peer.
	CrawlPeer(peer.AddrInfo) (*NodeKnows, error)

	// Stop shuts down the worker cleanly.
	Stop() error
}

// NodeKnows stores the collected addresses for a given ID
type NodeKnows struct {
	id            peer.ID
	knows         []peer.AddrInfo
	info          peerMetadata
	pluginResults map[string]PluginResult
}

// PluginResult encapsulates the result of calling a plugin on a peer.
type PluginResult struct {
	Error  error
	Result interface{}
}

type peerMetadata struct {
	AgentVersion           *string
	DHTProtocol            string
	CrawlStartedTimestamp  time.Time
	CrawlFinishedTimestamp time.Time
	SupportedProtocols     []string
}

// CrawlOutput is the output of a crawl.
type CrawlOutput struct {
	Nodes map[peer.ID]CrawledNode
}

// CrawledNode contains information about a single crawled node.
type CrawledNode struct {
	ID                     peer.ID
	MultiAddrs             []ma.Multiaddr
	Crawlable              bool
	AgentVersion           string
	Neighbours             []peer.ID
	CrawlStartedTimestamp  time.Time
	CrawlFinishedTimestamp time.Time
	SupportedProtocols     []string
	PluginData             map[string]PluginResult
}

type workerCrawlResult struct {
	node *NodeKnows
	err  error
}

// CrawlerConfig contains configuration for the crawler.
type CrawlerConfig struct {
	NumWorkers         int            `yaml:"num_workers"`
	BootstrapPeers     []string       `yaml:"bootstrap_peers"`
	ConcurrentRequests int            `yaml:"concurrent_requests"`
	WorkerConfig       WorkerConfig   `yaml:"worker_config"`
	Plugins            []PluginConfig `yaml:"plugins"`
}

// A CrawlManager manages crawling the network.
// It contains multiple workers, with a libp2p node each, which are used to
// execute requests concurrently.
type CrawlManager struct {
	resultChan       chan workerCrawlResult
	toCrawl          []peer.AddrInfo
	tokenBucket      chan int
	workers          []CrawlerWorker
	crawlsInProgress map[peer.ID]struct{}

	// We use this map not only to store whether we crawled a node but also to store a nodes multiaddresses
	crawled       map[peer.ID][]ma.Multiaddr
	knows         map[peer.ID][]peer.ID
	online        map[peer.ID]bool
	peerMetadata  map[peer.ID]peerMetadata
	pluginResults map[peer.ID]map[string]PluginResult
}

// NewCrawlManager creates a new CrawlManager.
// This attempts to create the specified number of workers and plugins, which
// may fail.
func NewCrawlManager(config CrawlerConfig, ph *PreimageHandler) (*CrawlManager, error) {
	cm := &CrawlManager{
		resultChan:       make(chan workerCrawlResult),
		tokenBucket:      make(chan int, config.NumWorkers*config.ConcurrentRequests),
		crawled:          make(map[peer.ID][]ma.Multiaddr),
		online:           make(map[peer.ID]bool),
		knows:            make(map[peer.ID][]peer.ID),
		peerMetadata:     make(map[peer.ID]peerMetadata),
		pluginResults:    make(map[peer.ID]map[string]PluginResult),
		crawlsInProgress: make(map[peer.ID]struct{}),
	}

	// Create workers
	for i := 0; i < config.NumWorkers; i++ {
		worker, err := NewLibp2pWorker(config.WorkerConfig, config.Plugins, ph)
		if err != nil {
			return nil, fmt.Errorf("unable to create worker: %w", err)
		}
		cm.workers = append(cm.workers, worker)
	}

	// Create concurrent work tokens, round-robin assign the workers by ID
	for i := 0; i < config.ConcurrentRequests; i++ {
		cm.tokenBucket <- i % config.NumWorkers
	}

	// Parse and add bootstrap peers to queue
	for _, maddr := range config.BootstrapPeers {
		pinfo, err := parsePeerString(maddr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse bootstrap peer address: %w", err)
		}
		cm.toCrawl = append(cm.toCrawl, *pinfo)
	}

	return cm, nil
}

// AddPeersToCrawl adds peers to the end of the queue.
// This must be called before CrawlNetwork.
// Not thread-safe.
func (cm *CrawlManager) AddPeersToCrawl(peers []peer.AddrInfo) {
	cm.toCrawl = append(cm.toCrawl, peers...)
}

// Stop shuts down all workers cleanly.
func (cm *CrawlManager) Stop() error {
	for _, worker := range cm.workers {
		err := worker.Stop()
		if err != nil {
			log.WithError(err).Warn("unable to stop worker")
		}
	}

	return nil
}

// CrawlNetwork crawls the network, starting at the configured bootstrap nodes.
// If any peers were added with AddPeersToCrawl, those will be asked, too.
// Apart from that, all nodes learned during the crawl will be contacted.
// Nodes are contacted only once, unless a previous connection attempt failed
// and new addresses have been learned since.
func (cm *CrawlManager) CrawlNetwork() *CrawlOutput {
	// Plan of action
	// 1. Add bootstraps to overflow
	// 2. Start dispatch loop
	//  2.1 get new nodes from resultChan and check if we need to crawl them, if yes: add to toCrawl
	//  2.2 if we can dispatch a crawl: dispatch from toCrawl
	//  2.3 break loop: idleTimer fired | (toCrawl empty && no request are out && knowQueue empty)
	//  return data
	log.Info("Starting crawl...")

	infoTicker := time.NewTicker(20 * time.Second)
	defer infoTicker.Stop()

	for {
		// check if we can break the loop
		if len(cm.toCrawl) == 0 &&
			len(cm.crawlsInProgress) == 0 {
			log.Info("Stopping crawl...")
			break
		}

		select {
		case report := <-cm.resultChan:
			// We have new information incoming
			if _, ok := cm.crawlsInProgress[report.node.id]; !ok {
				panic("received result for untracked crawl")
			}
			delete(cm.crawlsInProgress, report.node.id)
			node := report.node
			err := report.err
			if err != nil {
				log.WithFields(log.Fields{"Error": err}).Debug("Error while crawling")
				// TODO: Error handling
				// We still want to record at least the start/end timestamps
				if _, ok := cm.peerMetadata[node.id]; !ok {
					cm.peerMetadata[node.id] = node.info // TODO maybe merge instead of overwriting
				}
				continue
			}

			cm.online[node.id] = true
			cm.knows[node.id] = addrInfosToIDs(node.knows)
			cm.peerMetadata[node.id] = node.info // TODO: make the map merge together not overwrite each other
			cm.pluginResults[node.id] = node.pluginResults
			for _, p := range node.knows {
				cm.handleInputNodes(p)
			}
			log.WithFields(log.Fields{
				"Current Request": len(cm.crawlsInProgress),
				"toCrawl":         len(cm.toCrawl),
				"Reports":         len(cm.resultChan),
			}).Debug("Status of Manager")

		case id := <-cm.tokenBucket:
			// We can start a crawl, so let's do that
			if len(cm.toCrawl) > 0 {
				var node peer.AddrInfo
				node, cm.toCrawl = cm.toCrawl[0], cm.toCrawl[1:]

				// Check if we're already crawling that node
				if _, ok := cm.crawlsInProgress[node.ID]; ok {
					log.WithFields(log.Fields{"node": node.ID}).Debug("already being crawled, not dispatching crawl request")
					cm.tokenBucket <- id
				} else {
					log.WithFields(log.Fields{"node": node.ID}).Debug("dispatching crawl request")
					cm.crawlsInProgress[node.ID] = struct{}{}
					go cm.dispatch(node, id)
				}
			} else {
				// nothing to do; return token
				cm.tokenBucket <- id
				// Sleep a bit, because we're probably at the end of the crawl and not much is happening.
				time.Sleep(10 * time.Millisecond)
			}

		case <-infoTicker.C:
			log.WithFields(log.Fields{
				"discovered nodes":            len(cm.crawled),
				"available workers":           len(cm.tokenBucket),
				"requests in flight":          len(cm.crawlsInProgress),
				"to-crawl-queue":              len(cm.toCrawl),
				"connectable+crawlable nodes": len(cm.online),
			}).Info("Periodic info on crawl status")
		}
	}

	return cm.createReport()
}

func (cm *CrawlManager) dispatch(node peer.AddrInfo, id int) {
	worker := cm.workers[id]
	before := time.Now()
	result, err := worker.CrawlPeer(node) // FIXME: worker selection
	after := time.Now()
	if err != nil {
		log.WithError(err).WithField("peer", node).Debug("unable to crawl node")
	} else {
		log.WithField("Result", result).Debug("crawled node")
	}

	if result == nil {
		result = new(NodeKnows)
		result.id = node.ID
	}
	result.info.CrawlStartedTimestamp = before
	result.info.CrawlFinishedTimestamp = after

	cm.resultChan <- workerCrawlResult{node: result, err: err}
	cm.tokenBucket <- id
}

func (cm *CrawlManager) handleInputNodes(node peer.AddrInfo) {
	oldAddrs, crawled := cm.crawled[node.ID]
	_, online := cm.online[node.ID]
	if crawled && online {
		return
	}
	if crawled && !online {
		// Check if there are any new addresses. If so, connect to them
		newAddrs := filterOutOldAddresses(oldAddrs, stripLocalAddrs(node).Addrs)
		if len(newAddrs) == 0 {
			// Nothing new, don't bother dialing again
			return
		}
		log.WithFields(log.Fields{"node": node.ID}).Debug("Adding new addresses to crawled")
		cm.crawled[node.ID] = append(cm.crawled[node.ID], newAddrs...)
		workload := peer.AddrInfo{
			ID:    node.ID,
			Addrs: newAddrs,
		}
		log.WithFields(log.Fields{"node": node.ID}).Debug("Try new addresses")
		cm.toCrawl = append(cm.toCrawl, workload)
		return
	}

	// If not, we remember that we've seen it and add it to the work queue, so that a worker will eventually crawl it.
	cm.crawled[node.ID] = node.Addrs
	log.WithFields(log.Fields{"node": node.ID}).Debug("Adding newer seen node")
	cm.toCrawl = append(cm.toCrawl, node)
}

func (cm *CrawlManager) createReport() *CrawlOutput {
	// OutputFilePath a crawl report into the log
	log.WithFields(log.Fields{
		"number of nodes":   len(cm.crawled),
		"connectable nodes": len(cm.online),
	}).Info("Crawl finished. Summary of results.")

	out := CrawlOutput{Nodes: make(map[peer.ID]CrawledNode)}
	for node, Addresses := range cm.crawled {
		var status CrawledNode
		status.ID = node
		status.MultiAddrs = Addresses
		if online, found := cm.online[node]; found {
			status.Crawlable = online
		} else {
			status.Crawlable = false // Default value if not found
		}
		if neighbours, found := cm.knows[node]; found {
			status.Neighbours = neighbours
		} else {
			status.Neighbours = []peer.ID{}
		}

		if metadata, ok := cm.peerMetadata[node]; ok {
			if metadata.AgentVersion != nil {
				status.AgentVersion = *metadata.AgentVersion
			}
			status.CrawlStartedTimestamp = metadata.CrawlStartedTimestamp
			status.CrawlFinishedTimestamp = metadata.CrawlFinishedTimestamp
			status.SupportedProtocols = metadata.SupportedProtocols
		}

		if pluginResults, ok := cm.pluginResults[node]; ok {
			status.PluginData = pluginResults
		}

		out.Nodes[node] = status
	}
	return &out
}
