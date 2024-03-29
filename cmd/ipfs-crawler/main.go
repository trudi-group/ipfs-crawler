// Package main implements the ipfs-crawler executable.
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

	// Plugins
	_ "ipfs-crawler/plugins/bsprobe"
)

// Config is the configuration for the ipfs-crawler executable.
type Config struct {
	// Path to output directory.
	OutputDirectoryPath string `yaml:"output_directory_path"`

	// File where the nodes between crawls are cached (if caching is enabled).
	CacheFilePath *string `yaml:"cache_file_path"`

	// Settings for the crawler.
	CrawlOptions crawlLib.CrawlManagerConfig `yaml:"crawler"`
}

func main() {
	var debug bool
	var configFilePath string
	var help bool

	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.StringVar(&configFilePath, "config", "dist/config_ipfs.yaml", "path to the configuration file")
	flag.BoolVar(&help, "help", false, "print usage")
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
	log.WithField("path", config.OutputDirectoryPath).Info("writing results to")

	// Create crawl manager
	cm, err := crawlLib.NewCrawlManager(config.CrawlOptions)
	if err != nil {
		log.Fatal(fmt.Errorf("unable to set up crawler: %w", err))
	}
	log.Info("created crawl manager")

	// Add cached nodes if we have them
	if config.CacheFilePath != nil {
		cachedNodes, err := crawlLib.RestoreNodeCache(*config.CacheFilePath)
		if err != nil {
			// First time may fail
			log.WithError(err).Warn("unable to load cached peers, ignoring")
		} else {
			log.WithField("num", len(cachedNodes)).Info("loaded cached peers, adding to queue")
			cm.AddPeersToCrawl(cachedNodes)
		}
	} else {
		log.Info("node caching disabled")
	}

	// Start the crawl
	before := time.Now()
	beforeString := before.UTC().Format("2006-01-02_15-04-05_UTC")
	report := cm.CrawlNetwork()
	after := time.Now()

	// Stop libp2p nodes etc.
	log.Debug("stopping crawl manager")
	err = cm.Stop()
	if err != nil {
		log.WithError(err).Warn("unable to gracefully shut down")
	}
	log.Info("stopped crawl manager")

	// Write output
	log.Debug("writing node metadata")
	err = report.WriteMetadata(before, after, path.Join(config.OutputDirectoryPath, fmt.Sprintf("visitedPeers_%s.json", beforeString)))
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("writing peer graph")
	err = report.WritePeergraph(path.Join(config.OutputDirectoryPath, fmt.Sprintf("peerGraph_%s.csv", beforeString)))
	if err != nil {
		log.Fatal(err)
	}
	log.Info("wrote results")

	// Write node cache
	if config.CacheFilePath != nil {
		err = report.SaveNodeCache(*config.CacheFilePath)
		if err != nil {
			log.Fatal(fmt.Errorf("unable to save online nodes to cache: %w", err))
		}
		log.WithField("path", config.CacheFilePath).Info("saved online nodes to cache")
	}
}

func parseConfig(configFilePath string) (*Config, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %w", err)
	}

	var config Config
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %w", err)
	}

	return &config, nil
}
