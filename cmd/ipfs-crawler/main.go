package main

import (
	"net/http"

	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	utils "ipfs-crawler/common"
	crawlLib "ipfs-crawler/crawling"
	watcher "ipfs-crawler/watcher"
	// "os/signal"
	"strings"

	peer "github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type MainConfig struct {
    // Time format of log entries.
	LogTimeFormat string
    // Buffersize of each queue that we are using for communication between threads
	BufferSize     int
    // Log level. Debug contains a lot but is very spammy
	LogLevel      string
    // Indicates whether nodes should be cached during crawl runs to speed up the next successive crawl
	UseCache      bool
    // File where the nodes between crawls are cached (if caching is enabled)
	CacheFile     string
	 // Output Folder
	 Outpath string
	// Port on which prometheus metrics are exposed
	PrometheusMetricsPort int
	 PreImagePath string
	 NumPreImages int
}

type MonitorConfig struct {
    Enabled bool
    Input string
    Output string
}

const (
	// connectTimeout = 2 * time.Second
	// Indicates whether nodes should be cached during crawl runs to speed up the next successive crawl
	// useCache = true
	// File where the nodes between crawls are cached (if caching is enabled)
	// cacheFile = "nodes.cache"
	// Time format of log entries. Go, why you so ugly?
	// logTimeFormat = "15:04:05"
	// Log level. Debug contains a lot but is very spammy
	// logLevel = log.InfoLevel
	// Buffersize of each queue that we are using for communication between threads
)

// FOR TESTING PURPOSES: OUR LOCAL NODE
// "/ip4/127.0.0.1/tcp/4003/ipfs/QmamSnfS9bVjGgJJ57hznpCyMnesAtD3BidU8gfFBwUD7U", // local node

// TODO:
// * More robust error handling when connecting or receiving messages
// * Are relays used when connecting?
func init() {
    // Set up defaults
    // f, err := os.Create("memprofile")
    // if err != nil {
    //     log.Fatal("could not create memory profile: ", err)
    // }
    // defer f.Close()
    // pprof.WriteHeapProfile(f)

    viper.SetDefault("loglevel", "debug")
    viper.SetDefault("useCache", true)
    viper.SetDefault("cacheFile", "nodes.cache")
    viper.SetDefault("crawloptions.numworkers", 8)
    viper.SetDefault("logTimeFormat", "15:04:05")
    viper.SetDefault("logLevel", "info")
    viper.SetDefault("bufferSize", 64384)
    viper.SetDefault("prometheusMetricsPort", 2112)
	viper.SetDefault("PreImagePath", "precomputed_hashes/preimages.csv")
	viper.SetDefault("NumPreImages", 16777216)
    viper.SetDefault("module.monitor.enabled", false)

}

func main() {
	// There's a clash between libp2p (2024) and ipfs (512) minimum key lenghts -> set it to the one used in IPFS.
	// Since libp2p ist initialized earlier than our main() function we have to set it via the command line.

	// Setting up config
    var saveconfig string
    var configFile string
    var help bool
    // setup and bind flags to the viper config. Don't use flags default values they are thrown away, but we have to set them. Viper defaults are authoritative.
	flag.String("loglevel", "", "Set LogLevel")
    flag.String("cacheFile", "", "Set cache")
    flag.Bool("useCache", true, "Use cache")
    flag.String("OutPath", "", "Path for output")
    flag.String("PreImagePath", "", "Path to PreImageFile")
    flag.String("CanaryFile", "", "Path to canary file")
    flag.Bool("Sanity", true, "Use canary checks")
    flag.Bool("WriteToFile", true, "help message for flagname")
    //Create Flags for Bitswap monitoring module
    flag.Bool("UseMonitor", false, "activate BitSwap monitoring")
    flag.String("Cids", "", "Path to the monitored cids")
    flag.String("CidsOut","", "Path to the cid monitoring output file")
    // Setup flags which don't belong into the config
    flag.StringVar(&saveconfig, "saveconfig", "", "save current config to path")
    flag.StringVar(&configFile, "config", "", "Path to config file.")
    flag.BoolVar(&help, "help", false, "Print usage.")
    flag.Parse()
    viper.BindPFlag("loglevel",flag.Lookup("loglevel"))
    viper.BindPFlag("cacheFile",flag.Lookup("cacheFile"))
    viper.BindPFlag("useCache",flag.Lookup("useCache"))
    viper.BindPFlag("OutPath",flag.Lookup("OutPath"))
    viper.BindPFlag("PreImagePath",flag.Lookup("PreImagePath"))
    viper.BindPFlag("CanaryFile",flag.Lookup("CanaryFile"))
    viper.BindPFlag("Sanity",flag.Lookup("Sanity"))
    viper.BindPFlag("WriteToFileFlag",flag.Lookup("WriteToFile"))
    // viper binds for cid monitoring
    viper.BindPFlag("module.monitor.Enable", flag.Lookup("UseMonitor"))
    viper.BindPFlag("module.monitor.Input", flag.Lookup("Cids"))
    viper.BindPFlag("module.monitor.Output", flag.Lookup("CidsOut"))

	config := setupViper()
    if help {
        flag.PrintDefaults()
        os.Exit(0)
    }
    if saveconfig != "" {
        viper.WriteConfigAs(saveconfig)
    }

	// Set up prometheus handler
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", config.PrometheusMetricsPort), nil)

	// Setting up the logging
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = config.LogTimeFormat
	// formatter.DisableSorting = true
	// Don't truncate the levels
	formatter.DisableLevelTruncation = true
	log.SetFormatter(formatter)
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(logLevel)

    // get monitoring config
    var monitorcfg MonitorConfig
    err = viper.UnmarshalKey("module.monitor",&monitorcfg)
	if err != nil {
		log.WithField("error", err).Error("Cannot read monitoring config, disable for now")
	}
    // fmt.Println(monitorcfg)

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

	// Create the directory for output data, if it does not exist
	err = utils.CreateDirIfNotExists(config.Outpath)
	if err != nil {
		log.WithField("err", err).Error("Could not create output directory, crawl result will not be stored! Continue? (y/n)")
		if !utils.AskYesNo() {
			os.Exit(0)
		}
	}

	// Second, load preimageHandler
	cm := crawlLib.NewCrawlManagerV2(config.BufferSize)
	preimages, err := crawlLib.LoadPreimages(config.PreImagePath, config.NumPreImages)
	if err != nil {
		log.WithField("err", err).Error("Could not load pre-images. Continue anyway? (y/n)")
		if !utils.AskYesNo() {
			os.Exit(0)
		}
	}
	handler := crawlLib.PreImageHandler{
		PreImages: preimages,
	}
    // create BitSwap Pipeline
    var bsmanager *watcher.BSManager
    bsmanager = watcher.NewBSManager()
    bscon, bscancel := context.WithCancel(context.Background())
    if monitorcfg.Enabled {
        cids := watcher.ReadCids(monitorcfg.Input)
        bsmanager.AddCid(cids)
        go bsmanager.Start(bscon)
    }

    // Create the crawl-workers
    numWorkers := viper.GetInt("crawloptions.numworkers")
	log.WithField("numberOfWorkers", numWorkers).Info("Creating workers...")
	for i := 0; i < numWorkers; i++ {
		worker := crawlLib.NewIPFSWorker(0, context.Background())
		worker.AddPreimages(&handler)
        // if use monitor: create monitoring worker with foreign host, and connect monitoring worker with connected event
        if monitorcfg.Enabled {
            bsworker, _ := watcher.NewBSWorker(worker.GetHost())
            bsmanager.AddWorker(bsworker)
            worker.Events.Subscribe("connected", bsworker)
        }
		cm.AddWorker(worker)
	}

	// Load bootstrappeers from config
	var bs []string
	viper.UnmarshalKey("crawloptions.bootstrappeers", &bs)

	// Convert the strings to node.AddrInfos
	var bootstrappeers []*peer.AddrInfo
	for _, maddr := range bs {
		pinfo, err := utils.ParsePeerString(maddr)
		if err != nil {
			log.WithFields(log.Fields{
				"multiaddr": maddr,
				"err": err,
			}).Warning("Error parsing bootstrap peers.")
		}
		bootstrappeers = append(bootstrappeers, pinfo)
	}
	
	// Add cached nodes if we have them
	if config.UseCache {
		cachedNodes, err := crawlLib.RestoreNodeCache(config.CacheFile)
		if err == nil {
			log.WithField("amount", len(cachedNodes)).Info("Adding cached peer to crawl queue.")
			bootstrappeers = append(bootstrappeers, cachedNodes...)
		}
	}
	
	// Start the crawl
	report := cm.CrawlNetwork(bootstrappeers)
	startStamp := report.StartDate
	endStamp := report.EndDate
	crawlLib.ReportToFile(report, config.Outpath + fmt.Sprintf("visitedPeers_%s_%s.json", startStamp, endStamp))
	crawlLib.WritePeergraph(report, config.Outpath + fmt.Sprintf("peerGraph_%s_%s.csv", startStamp, endStamp))
	if config.UseCache{
		crawlLib.SaveNodeCache(report, config.CacheFile)
		log.WithField("File", config.CacheFile).Info("Online nodes saved in cache")
	}
    if monitorcfg.Enabled {
        bscancel()
        bsmanager.Wait()
        cidLog := bsmanager.GetReport()
        watcher.ToJson(cidLog, monitorcfg.Output)
    }
	// for _, p := range bootstrappeers {
	// 	log.WithField("addr", p).Debug("Adding bootstrap peer to crawl queue.")
	// 	// fmt.Printf("[%s] Adding bootstrap peer to crawl queue: %s\n", Timestamp(), ainfo)
	// 	cm.InputQueue <- *p
	// }

	// Catch strg+c
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// go func() {
	// 	for sig := range c {
	// 		fmt.Println(sig)
	// 		// ShutDown(cm)
	// 	}
	// }()
	// exit := <-cm.Done
	// Fuck you go compiler
	// _ = exit
	// log.WithFields(log.Fields{
	// 	"inputQueueLength": cm.GetInputQueueLen(),
	// 	"workQueueLength":  cm.GetWorkQueueLen(),
	// }).Info("Exit successful.")
	os.Exit(0)
}

func setupViper() MainConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		// panic(fmt.Errorf("Fatal error config file: %s \n", err))
		log.Warning(err)
	}
	// write read config back to config obj
	var config MainConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
	return config
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
	var bootstrappeers []*peer.AddrInfo
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
		bootstrappeers = append(bootstrappeers, ainfo)
	}

	return bootstrappeers, nil
}
