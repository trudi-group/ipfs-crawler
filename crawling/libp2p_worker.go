package crawling

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	log "github.com/sirupsen/logrus"
)

// DesyncMillisMax sets a limit on the random backoff performed before each
// request to de-sync.
const DesyncMillisMax = 500

// The WorkerConfig configures a single worker.
type WorkerConfig struct {
	ConnectTimeout     time.Duration `yaml:"connect_timeout"`
	ConnectionAttempts uint          `yaml:"connection_attempts"`
	UserAgent          string        `yaml:"user_agent"`
}

func (c WorkerConfig) check() error {
	if c.ConnectTimeout <= time.Duration(0) {
		return fmt.Errorf("missing connection timeout")
	}
	if c.ConnectionAttempts == 0 {
		return fmt.Errorf("invalid or missing connection attempts")
	}
	if len(c.UserAgent) == 0 {
		return fmt.Errorf("missing user agent")
	}
	return nil
}

// A Libp2pWorker implements the worker interface for a libp2p host.
type Libp2pWorker struct {
	host        host.Host
	config      WorkerConfig
	crawler     *crawler
	plugins     []Plugin
	closed      chan struct{}
	closingLock sync.Mutex
}

// NewLibp2pWorker creates a new libp2p worker.
// This initializes a new libp2p host with a unique keypair, configures the
// libp2p resource manager to be disabled, and initializes all given plugins on
// the host.
func NewLibp2pWorker(config WorkerConfig, pluginConfigs []PluginConfig, preimageHandler *PreimageHandler, crawlerConfig CrawlerConfig) (*Libp2pWorker, error) {
	err := config.check()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	w := &Libp2pWorker{
		config: config,
		closed: make(chan struct{}),
	}

	rm := network.NullResourceManager{}
	cm := connmgr.NullConnMgr{}

	// Create libp2p host
	opts := []libp2p.Option{
		libp2p.ResourceManager(&rm), libp2p.ConnectionManager(cm), libp2p.UserAgent(config.UserAgent), libp2p.DisableMetrics(),
		libp2p.SwarmOpts(swarm.WithReadOnlyBlackHoleDetector()),
		libp2p.UDPBlackHoleSuccessCounter(nil),
		libp2p.IPv6BlackHoleSuccessCounter(nil),
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create libp2p host: %w", err)
	}
	w.host = h

	// Create crawler "plugin"
	c, err := newCrawler(h, crawlerConfig, preimageHandler)
	if err != nil {
		return nil, fmt.Errorf("unable to create crawler plugin: %w", err)
	}
	w.crawler = c

	// Create plugins
	plugins, err := PluginsFromPluginConfigs(h, pluginConfigs)
	if err != nil {
		return nil, fmt.Errorf("unable to create plugins: %w", err)
	}
	w.plugins = plugins

	return w, nil
}

// connect attempts to open a connection to the given peer and
// waits for the Identify protocol to finish.
func (w *Libp2pWorker) connect(p peer.AddrInfo) (network.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.ConnectTimeout)
	defer cancel()

	// This internally calls w.host.Network().DialPeer(...) and then waits
	// for the identity protocol to finish.
	err := w.host.Connect(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	// What we really want is the connection itself, though.
	// So we just call it again...
	ctx, cancel = context.WithTimeout(context.Background(), w.config.ConnectTimeout)
	defer cancel()
	c, err := w.host.Network().DialPeer(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return c, nil
}

// CrawlPeer implements worker.
func (w *Libp2pWorker) crawlPeer(remote peer.AddrInfo) (*rawNodeInformation, error) {
	// Sleep to de-sync
	time.Sleep(time.Duration(rand.Intn(DesyncMillisMax)) * time.Millisecond)

	// Connect to peer
	var conn network.Conn
	var err error
	for i := uint(0); i < w.config.ConnectionAttempts; i++ {
		conn, err = w.connect(remote)
		if err != nil {
			log.WithFields(log.Fields{
				"err":      err,
				"try":      i + 1,
				"destAddr": remote,
			}).Debug("could not connect")
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Execute crawler "plugin"
	crawlBeginTs := time.Now()
	crawlData, crawlErr := w.crawler.HandlePeer(remote)
	crawlEndTs := time.Now()
	if crawlErr != nil {
		log.WithError(crawlErr).WithField("peer", remote.ID).Debug("unable to crawl peer")
	}

	// Execute plugins
	pluginResults := make(map[string]pluginResult)
	for _, p := range w.plugins {
		log.WithField("remote", remote.ID).WithField("plugin", p.Name()).Debug("executing plugin")
		res, err := p.HandlePeer(remote)
		if err != nil {
			log.WithError(err).WithField("remote", remote.ID).WithField("plugin", p.Name()).Debug("plugin failed")
		}
		pluginResults[p.Name()] = pluginResult{
			err:    err,
			result: res,
		}
	}

	// TODO figure out a way to actually _force_ identify a connection, potentially with retries.
	// We could call (*idService).identifyConn(c network.Conn), which we need to get via reflection or so first...
	// This seems fine for now. If the connection works, it's identified
	// (confirmed from testing).

	var infos peerMetadata
	agentVersion, err := w.host.Peerstore().Get(remote.ID, "AgentVersion")
	if err != nil {
		log.WithError(err).WithField("peer", remote.ID).Debug("unable to get agent version")
	} else {
		infos.AgentVersion = agentVersion.(string)
	}
	protocols, err := w.host.Peerstore().GetProtocols(remote.ID)
	if err != nil {
		log.WithError(err).WithField("peer", remote.ID).Warn("unable to get supported protocols")
	} else {
		infos.SupportedProtocols = protocols
	}

	return &rawNodeInformation{
		info: infos,
		crawlData: crawlResult{
			beginTimestamp: crawlBeginTs,
			endTimestamp:   crawlEndTs,
			err:            crawlErr,
			result:         crawlData,
		},
		pluginResults: pluginResults,
	}, nil
}

// Stop stops the Libp2pWorker.
// This shuts down any plugins and stops the libp2p host.
func (w *Libp2pWorker) stop() error {
	w.closingLock.Lock()
	select {
	case <-w.closed:
		// Already closed
		w.closingLock.Unlock()
		return nil
	default:
		// Not closed
		close(w.closed)
	}
	w.closingLock.Unlock()

	// Close crawler
	err := w.crawler.Shutdown()
	if err != nil {
		return fmt.Errorf("unable to shut down crawler: %w", err)
	}

	// Close plugins
	for _, p := range w.plugins {
		err := p.Shutdown()
		if err != nil {
			return fmt.Errorf("unable to shut down plugin %s: %w", p.Name(), err)
		}
	}

	// Close libp2p host
	err = w.host.Close()
	if err != nil {
		return fmt.Errorf("unable to close libp2p host: %w", err)
	}

	return nil
}
