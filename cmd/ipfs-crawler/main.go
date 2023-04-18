package main

import (
	"fmt"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	crawlLib "ipfs-crawler/crawling"
)

type MainConfig struct {
	// Path to output directory.
	OutputDirectoryPath string `yaml:"output_directory_path"`

	// Path to the preimage file.
	PreimageFilePath string `yaml:"preimage_file_path"`

	// File where the nodes between crawls are cached (if caching is enabled).
	CacheFilePath *string `yaml:"cache_file_path"`

	// Settings for the crawler.
	CrawlOptions crawlLib.CrawlerConfig `yaml:"crawler"`

	// Settings for the Bitswap Monitor.
	BitswapMonitor *BitswapMonitorConfig `yaml:"bitswap_monitor"`
}

type BitswapMonitorConfig struct {
	InputFilePath  string `yaml:"input_file_path"`
	OutputFilePath string `yaml:"output_file_path"`
}

func main() {
	var debug bool
	var configFilePath string
	var help bool

	flag.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	flag.StringVar(&configFilePath, "config", "dist/config_ipfs.yaml", "path to the configuration file")
	flag.BoolVar(&help, "help", false, "Print usage.")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Set up logging
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	log.SetFormatter(formatter)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	config, err := parseConfig(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Let's go!
	log.Info("Thank you for running our IPFS Crawler!")

	// First, check whether the weak RSA keys environment variable is set
	// There's a clash between libp2p (2024) and ipfs (512) minimum key sizes -> set it to the one used in IPFS.
	// Since libp2p ist initialized earlier than our main() function we have to set it via the command line.
	_, weakKeysAllowed := os.LookupEnv("LIBP2P_ALLOW_WEAK_RSA_KEYS")
	log.WithField("weak_RSA_keys", weakKeysAllowed).Debug("Checking whether weak RSA keys are allowed...")
	if !weakKeysAllowed {
		log.Fatal("Weak RSA keys are *disabled*. This is required to connect to most nodes. Set LIBP2P_ALLOW_WEAK_RSA_KEYS.")
	}

	// Create the directory for output data, if it does not exist
	err = os.MkdirAll(config.OutputDirectoryPath, 0o777)
	if err != nil {
		log.Fatal(fmt.Errorf("unable to create output directory: %w", err))
	}

	// Load preimageHandler
	preimageHandler, err := crawlLib.LoadPreimages(config.PreimageFilePath)
	if err != nil {
		log.Fatal(fmt.Errorf("unable to load preimages: %w", err))
	}

	/*
		// create BitSwap Pipeline
		var bscancel context.CancelFunc
		var bsmanager *watcher.BSManager
		if config.BitswapMonitor != nil {
			bsmanager = watcher.NewBSManager()
			var bscon context.Context
			bscon, bscancel = context.WithCancel(context.Background())
			cids := watcher.ReadCids(config.BitswapMonitor.InputFilePath)
			bsmanager.AddCid(cids)
			go bsmanager.Start(bscon)
		} else {
			log.Info("Bitswap Monitoring disabled")
		}
	*/

	// Create crawl manager
	em := crawlLib.NewEventManager()
	cm, err := crawlLib.NewCrawlManager(config.CrawlOptions, em, preimageHandler)
	if err != nil {
		log.Fatal(fmt.Errorf("unable to set up crawler: %w", err))
	}

	// Add cached nodes if we have them
	if config.CacheFilePath != nil {
		cachedNodes, err := crawlLib.RestoreNodeCache(*config.CacheFilePath)
		if err != nil {
			log.Fatal(fmt.Errorf("unable to load cached peers: %w", err))
		}
		log.WithField("amount", len(cachedNodes)).Info("Adding cached peers to crawl queue.")
		cm.AddPeersToCrawl(cachedNodes)
	}

	// Start the crawl
	before := time.Now()
	beforeString := before.UTC().Format("2006-01-02_15-04-05_UTC")
	report := cm.CrawlNetwork()
	after := time.Now()

	// Stop libp2p nodes etc.
	err = cm.Stop()
	if err != nil {
		log.WithError(err).Warn("unable to gracefully shut down")
	}

	// Write output
	err = crawlLib.ReportToFile(report, before, after, path.Join(config.OutputDirectoryPath, fmt.Sprintf("visitedPeers_%s.json", beforeString)))
	if err != nil {
		log.Fatal(err)
	}
	err = crawlLib.WritePeergraph(report, path.Join(config.OutputDirectoryPath, fmt.Sprintf("peerGraph_%s.csv", beforeString)))
	if err != nil {
		log.Fatal(err)
	}

	// Write node cache
	if config.CacheFilePath != nil {
		err = crawlLib.SaveNodeCache(report, *config.CacheFilePath)
		if err != nil {
			log.Fatal(err)
		}
		log.WithField("File", config.CacheFilePath).Info("Online nodes saved in cache")
	}
}

func parseConfig(configFilePath string) (*MainConfig, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %w", err)
	}

	var config MainConfig
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return &config, nil
}
