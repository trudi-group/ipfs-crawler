package crawling

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// CrawlOutput is the output of a crawl.
type CrawlOutput struct {
	nodes    map[peer.ID]nodeCrawlStatus
	addrInfo map[peer.ID][]ma.Multiaddr
}

// CrawlManagerConfig contains configuration for the crawl manager.
type CrawlManagerConfig struct {
	// Path to the preimage file.
	PreimageFilePath string `yaml:"preimage_file_path"`

	NumWorkers         uint           `yaml:"num_workers"`
	BootstrapPeers     []string       `yaml:"bootstrap_peers"`
	ConcurrentRequests uint           `yaml:"concurrent_requests"`
	WorkerConfig       WorkerConfig   `yaml:"worker_config"`
	Plugins            []PluginConfig `yaml:"plugins"`
	CrawlerConfig      CrawlerConfig  `yaml:"crawler_config"`
}

func (c *CrawlManagerConfig) check() error {
	if len(c.PreimageFilePath) == 0 {
		return fmt.Errorf("missing preimage file path")
	}
	if c.NumWorkers == 0 {
		return fmt.Errorf("missing or invalid num_workers")
	}
	if len(c.BootstrapPeers) == 0 {
		return fmt.Errorf("missing bootstrap peers")
	}
	if c.ConcurrentRequests == 0 {
		return fmt.Errorf("missing or invalid concurrent_requests")
	}
	return nil
}

// toCrawlQueue keeps track of which peers we need to crawl and what addresses
// they have.
// It also knows if we should potentially re-crawl a peer because of address
// changes since the last time we crawled.
type toCrawlQueue struct {
	queue    []peer.ID
	inQueue  map[peer.ID]struct{}
	addrInfo map[peer.ID][]ma.Multiaddr
}

// numPeers returns the number of peers we know about.
func (q *toCrawlQueue) numPeers() int {
	return len(q.addrInfo)
}

// len returns the length of the queue.
func (q *toCrawlQueue) len() int {
	return len(q.inQueue)
}

// pop removes the next item from the queue.
// panics if the queue is empty.
func (q *toCrawlQueue) pop() peer.AddrInfo {
	if q.len() == 0 {
		panic("empty queue")
	}

	var id peer.ID
	id, q.queue = q.queue[0], q.queue[1:]
	addr := q.addrInfo[id]
	delete(q.inQueue, id)

	return peer.AddrInfo{
		ID:    id,
		Addrs: addr,
	}
}

// push adds the peer's addresses to the cache and, if necessary, to the crawl
// queue.
func (q *toCrawlQueue) push(p peer.AddrInfo, force bool) {
	if force {
		// Just add it
		q.queue = append(q.queue, p.ID)
		q.inQueue[p.ID] = struct{}{}
		newAddrs := filterOutOldAddresses(q.addrInfo[p.ID], stripLocalAddrs(p.Addrs))
		q.addrInfo[p.ID] = append(q.addrInfo[p.ID], newAddrs...)
		return
	}

	oldAddrs, ok := q.addrInfo[p.ID]
	if !ok {
		// Not known at all, just add
		q.queue = append(q.queue, p.ID)
		q.inQueue[p.ID] = struct{}{}
		q.addrInfo[p.ID] = p.Addrs
		return
	}

	// Already in the queue or previously crawled, but maybe new addresses
	newAddrs := filterOutOldAddresses(oldAddrs, stripLocalAddrs(p.Addrs))
	if len(newAddrs) == 0 {
		// No new addresses, nothing to do
		return
	}

	// Add new addresses
	q.addrInfo[p.ID] = append(q.addrInfo[p.ID], newAddrs...)

	// If not in queue, re-add (with new addresses)
	if _, ok := q.inQueue[p.ID]; !ok {
		q.inQueue[p.ID] = struct{}{}
		q.queue = append(q.queue, p.ID)
	}
}

// A worker is a libp2p host which is used to crawl the network.
// It should support concurrent crawls.
// It should also execute any plugins on connectable nodes.
type worker interface {
	// crawlPeer crawls the given peer.
	crawlPeer(peer.AddrInfo) (*rawNodeInformation, error)

	// stop shuts down the worker cleanly.
	stop() error
}

// nodeCrawlResult is the result of probing a peer.
// The fields err and node are mutually exclusive.
type nodeCrawlResult struct {
	id      peer.ID
	startTs time.Time
	endTs   time.Time
	err     error
	node    *rawNodeInformation
}

// rawNodeInformation stores all information from probing a peer
type rawNodeInformation struct {
	info          peerMetadata
	crawlData     crawlResult
	pluginResults map[string]pluginResult
}

// crawlResult encapsulates the result of trying to crawl a peer.
// The fields err and result are mutually exclusive.
type crawlResult struct {
	beginTimestamp time.Time
	endTimestamp   time.Time
	err            error
	result         *crawlData
}

// crawlData contains the data obtained through crawling a peer, notably its
// neighborhood.
type crawlData struct {
	neighbors              []peer.AddrInfo
	crawlStartedTimestamp  time.Time
	crawlFinishedTimestamp time.Time
}

// pluginResult encapsulates the result of calling a plugin on a peer.
// The fields err and result are mutually exclusive.
type pluginResult struct {
	beginTimestamp time.Time
	endTimestamp   time.Time
	err            error
	result         interface{}
}

// nodeCrawlStatus is our knowledge of a peer, after trying to probe it at least
// once.
// The fields err and result are mutually exclusive.
type nodeCrawlStatus struct {
	startTs time.Time
	endTs   time.Time
	err     error
	result  *nodeInformation
}

// nodeInformation holds any information we know about a node.
// Most notably, this does not store addresses of DHT neighbors, because they
// are potentially big.
// The fields crawlDataError and crawlNeighbors are mutually
// exclusive.
type nodeInformation struct {
	info          peerMetadata
	pluginResults map[string]pluginResult

	crawlDataError   error
	crawlDataBeginTs time.Time
	crawlDataEndTs   time.Time
	crawlNeighbors   []peer.ID
}

type peerMetadata struct {
	AgentVersion string

	SupportedProtocols []protocol.ID
}

// A CrawlManager manages crawling the network.
// It contains multiple workers, with a libp2p node each, which are used to
// execute requests concurrently.
type CrawlManager struct {
	resultChan  chan nodeCrawlResult
	tokenBucket chan int
	workers     []worker

	crawlsInProgress map[peer.ID]struct{}
	crawled          map[peer.ID]nodeCrawlStatus
	toCrawl          *toCrawlQueue
}

// NewCrawlManager creates a new CrawlManager.
// This attempts to create the specified number of workers and plugins, which
// may fail.
func NewCrawlManager(config CrawlManagerConfig) (*CrawlManager, error) {
	err := config.check()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Load preimageHandler
	preimageHandler, err := LoadPreimages(config.PreimageFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to load preimages: %w", err)
	}
	log.WithField("path", config.PreimageFilePath).WithField("num", len(preimageHandler.preimages)).Info("loaded preimages")

	cm := &CrawlManager{
		resultChan:       make(chan nodeCrawlResult),
		tokenBucket:      make(chan int, config.NumWorkers*config.ConcurrentRequests),
		crawled:          make(map[peer.ID]nodeCrawlStatus),
		crawlsInProgress: make(map[peer.ID]struct{}),
		toCrawl: &toCrawlQueue{
			queue:    nil,
			addrInfo: make(map[peer.ID][]ma.Multiaddr),
			inQueue:  make(map[peer.ID]struct{}),
		},
	}

	// Create workers
	for i := uint(0); i < config.NumWorkers; i++ {
		worker, err := NewLibp2pWorker(config.WorkerConfig, config.Plugins, preimageHandler, config.CrawlerConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to create worker: %w", err)
		}
		cm.workers = append(cm.workers, worker)
	}

	// Create concurrent work tokens, round-robin assign the workers by ID
	for i := uint(0); i < config.ConcurrentRequests; i++ {
		cm.tokenBucket <- int(i % config.NumWorkers)
	}

	// Parse and add bootstrap peers to queue
	for _, maddr := range config.BootstrapPeers {
		pinfo, err := parsePeerString(maddr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse bootstrap peer address: %w", err)
		}
		cm.toCrawl.push(*pinfo, false)
	}

	return cm, nil
}

// AddPeersToCrawl adds peers to the end of the queue.
// This must be called before CrawlNetwork.
func (cm *CrawlManager) AddPeersToCrawl(peers []peer.AddrInfo) {
	for _, p := range peers {
		cm.toCrawl.push(p, false)
	}
}

// Stop shuts down all workers cleanly.
func (cm *CrawlManager) Stop() error {
	for _, worker := range cm.workers {
		err := worker.stop()
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
func (cm *CrawlManager) CrawlNetwork() CrawlOutput {
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

	for cm.toCrawl.len() != 0 ||
		len(cm.crawlsInProgress) != 0 {

		select {
		case report := <-cm.resultChan:
			// We have new information incoming
			if _, ok := cm.crawlsInProgress[report.id]; !ok {
				panic("received result for untracked crawl")
			}
			delete(cm.crawlsInProgress, report.id)

			// Insert into our "database"
			cm.upsertCrawlResult(report)

			if report.err != nil {
				log.WithFields(log.Fields{"Error": report.err}).Debug("Error while crawling")
				continue
			}

			// Add new peers to queue
			if report.node.crawlData.result != nil {
				for _, addrInfo := range report.node.crawlData.result.neighbors {
					cm.handleNewNode(addrInfo)
				}
			}

			log.WithFields(log.Fields{
				"Current Request": len(cm.crawlsInProgress),
				"toCrawl":         cm.toCrawl.len(),
				"Reports":         len(cm.resultChan),
			}).Debug("Status of Manager")

		case id := <-cm.tokenBucket:
			// We have an available worker
			if cm.toCrawl.len() > 0 {
				node := cm.toCrawl.pop()

				// Check if we're already crawling that node
				if _, ok := cm.crawlsInProgress[node.ID]; ok {
					log.WithFields(log.Fields{"node": node.ID}).Debug("already being crawled, not dispatching crawl request")

					// Return to queue, maybe the crawl fails
					cm.toCrawl.push(node, true)
					cm.tokenBucket <- id
				} else {
					// Check if we crawled the node already
					if state, ok := cm.crawled[node.ID]; !ok || (ok && state.err != nil) || (ok && state.err == nil && state.result.crawlDataError != nil) {
						log.WithFields(log.Fields{"node": node.ID}).Debug("dispatching crawl request")
						cm.crawlsInProgress[node.ID] = struct{}{}
						go cm.dispatch(node, id)
					} else {
						log.WithFields(log.Fields{"node": node.ID}).Debug("already crawled, not dispatching crawl request")
						cm.tokenBucket <- id
					}
				}
			} else {
				// nothing to do; return token
				cm.tokenBucket <- id
				// Sleep a bit, because we're probably at the end of the crawl and not much is happening.
				time.Sleep(10 * time.Millisecond)
			}

		case <-infoTicker.C:
			numConnectable := 0
			numCrawlable := 0
			for _, status := range cm.crawled {
				if status.err == nil {
					numConnectable++
					if status.result.crawlDataError == nil {
						numCrawlable++
					}
				}
			}
			log.WithFields(log.Fields{
				"discovered nodes":            cm.toCrawl.numPeers(),
				"available workers":           len(cm.tokenBucket),
				"requests in flight":          len(cm.crawlsInProgress),
				"to-crawl-queue":              cm.toCrawl.len(),
				"connectable nodes":           numConnectable,
				"connectable+crawlable nodes": numCrawlable,
			}).Info("Periodic info on crawl status")
		}
	}

	return cm.createReport()
}

func (cm *CrawlManager) upsertCrawlResult(report nodeCrawlResult) {
	// TODO maybe modify existing entry with new information?
	ncs := nodeCrawlStatus{
		result:  nil,
		startTs: report.startTs,
		endTs:   report.endTs,
		err:     report.err,
	}
	if report.node != nil {
		ncs.result = new(nodeInformation)
		ncs.result.pluginResults = report.node.pluginResults
		ncs.result.info = report.node.info
		ncs.result.crawlDataError = report.node.crawlData.err
		ncs.result.crawlDataBeginTs = report.node.crawlData.beginTimestamp
		ncs.result.crawlDataEndTs = report.node.crawlData.endTimestamp
		if report.node.crawlData.result != nil {
			for _, p := range report.node.crawlData.result.neighbors {
				ncs.result.crawlNeighbors = append(ncs.result.crawlNeighbors, p.ID)
			}
		}
	}
	cm.crawled[report.id] = ncs
}

func (cm *CrawlManager) dispatch(node peer.AddrInfo, id int) {
	worker := cm.workers[id]
	before := time.Now()
	result, err := worker.crawlPeer(node)
	after := time.Now()
	if err != nil {
		log.WithError(err).WithField("peer", node).Debug("unable to crawl node")
	} else {
		log.WithField("Result", result).Debug("crawled node")
	}

	cm.resultChan <- nodeCrawlResult{
		id:      node.ID,
		node:    result,
		startTs: before,
		endTs:   after,
		err:     err,
	}
	cm.tokenBucket <- id
}

func (cm *CrawlManager) handleNewNode(node peer.AddrInfo) {
	state, ok := cm.crawled[node.ID]
	if ok {
		if state.err == nil && state.result.crawlDataError == nil {
			// We've crawled the node successfully before, no need to try again.
			return
		}
	}

	// We've either not crawled the node or failed before.
	// The queue will decide whether we have new addresses and should retry.
	cm.toCrawl.push(node, false)
}

func (cm *CrawlManager) createReport() CrawlOutput {
	numNodes := 0
	numConnectable := 0
	numCrawlable := 0

	for _, state := range cm.crawled {
		numNodes++
		if state.err == nil {
			numConnectable++
			if state.result.crawlDataError == nil {
				numCrawlable++
			}
		}
	}

	log.WithFields(log.Fields{
		"number of nodes":   numNodes,
		"connectable nodes": numConnectable,
		"crawlable nodes":   numCrawlable,
	}).Info("Crawl finished. Summary of results.")

	return CrawlOutput{
		nodes:    cm.crawled,
		addrInfo: cm.toCrawl.addrInfo,
	}
}
