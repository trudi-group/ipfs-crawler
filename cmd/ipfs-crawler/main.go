package main

import (
	utils "ipfs-crawler/common"
	crawlLib "ipfs-crawler/crawling"

	"bufio"
	"fmt"
	"os"
	// "os/signal"
	"strings"
	"context"

	peer "github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
    flag "github.com/spf13/pflag"
)

type MainConfig struct {
    // Number of Workers attached.
	NumWorker     int
    // Time format of log entries.
	LogTimeFormat string
    // File which contains the bootstrap peers
	BootstrapFile string
    // Buffersize of each queue that we are using for communication between threads
	QueueSize     int
    // Log level. Debug contains a lot but is very spammy
	LogLevel      string
    // Indicates whether nodes should be cached during crawl runs to speed up the next successive crawl
	UseCache      bool
    // File where the nodes between crawls are cached (if caching is enabled)
	CacheFile     string
	 // Output Folder
	 Outpath string
}

const (
	// numWorkers = 1
	// connectTimeout = 2 * time.Second
	// Indicates whether nodes should be cached during crawl runs to speed up the next successive crawl
	// useCache = true
	// File where the nodes between crawls are cached (if caching is enabled)
	// cacheFile = "nodes.cache"
	// Time format of log entries. Go, why you so ugly?
	// logTimeFormat = "15:04:05"
	// Log level. Debug contains a lot but is very spammy
	// logLevel = log.InfoLevel
	// File which contains the bootstrap peers
	// bootstrapFile = "configs/bootstrappeers.txt"
	// Buffersize of each queue that we are using for communication between threads
	// queueSize = 64384
)

// FOR TESTING PURPOSES: OUR LOCAL NODE
// "/ip4/127.0.0.1/tcp/4003/ipfs/QmamSnfS9bVjGgJJ57hznpCyMnesAtD3BidU8gfFBwUD7U", // local node

// TODO:
// * More robust error handling when connecting or receiving messages
// * Are relays used when connecting?
func init() {
    // Set up defaults
    viper.SetDefault("loglevel", "debug")
    viper.SetDefault("useCache", true)
    viper.SetDefault("cacheFile", "nodes.cache")
    viper.SetDefault("numWorker", 1)
    viper.SetDefault("logTimeFormat", "15:04:05")
    viper.SetDefault("bootstrapFile", "configs/bootstrappeers.txt")
    viper.SetDefault("logLevel", "info")
    viper.SetDefault("queueSize", 64384)
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
    flag.String("bootstrapFile", "", "Path to bootstrapsfile")
    flag.String("cacheFile", "", "Set cache")
    flag.Bool("useCache", true, "Use cache")
    flag.String("OutPath", "", "Path for output")
    flag.String("PreImagePath", "", "Path to PreImageFile")
    flag.String("CanaryFile", "", "Path to canary file")
    flag.Bool("Sanity", true, "Use canary checks")
    flag.Bool("WriteToFile", true, "help message for flagname")
    // Setup flags which don't belong into the config
    flag.StringVar(&saveconfig, "saveconfig", "", "save current config to path")
    flag.StringVar(&configFile, "config", "", "Path to config file.")
    flag.BoolVar(&help, "help", false, "Print usage.")
    flag.Parse()
    viper.BindPFlag("loglevel",flag.Lookup("loglevel"))
    viper.BindPFlag("bootstrapFile",flag.Lookup("bootstrapFile"))
    viper.BindPFlag("cacheFile",flag.Lookup("cacheFile"))
    viper.BindPFlag("useCache",flag.Lookup("useCache"))
    viper.BindPFlag("OutPath",flag.Lookup("OutPath"))
    viper.BindPFlag("PreImagePath",flag.Lookup("PreImagePath"))
    viper.BindPFlag("CanaryFile",flag.Lookup("CanaryFile"))
    viper.BindPFlag("Sanity",flag.Lookup("Sanity"))
    viper.BindPFlag("WriteToFileFlag",flag.Lookup("WriteToFile"))

	config := setupViper()
    if help {
        flag.PrintDefaults()
        os.Exit(0)
    }
    if saveconfig != "" {
        viper.WriteConfigAs(saveconfig)
    }


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

	// Second, check if the pre-image file exists
	cm := crawlLib.NewCrawlManagerV2(config.QueueSize)
	log.WithField("numberOfWorkers", config.NumWorker).Info("Creating workers...")
	// for i := 0; i < config.NumWorker; i++ {
	// 	cm.CreateAndAddWorker()
	// }
	worker := crawlLib.NewIPFSWorker(0, context.Background())
	cm.AddWorker(worker)

	bootstrappeers, err := LoadBootstrapList(config.BootstrapFile)
	if err != nil {
		panic(err)
	}
	if config.UseCache {
		cachedNodes, err := crawlLib.RestoreNodeCache(config.CacheFile)
		if err == nil {
			log.WithField("amount", len(cachedNodes)).Info("Adding cached peer to crawl queue.")
			bootstrappeers = append(bootstrappeers, cachedNodes...)
		}
	}
	report := cm.CrawlNetwork(bootstrappeers)
	startStamp := report.StartDate
	endStamp := report.EndDate
	crawlLib.ReportToFile(report, config.Outpath + fmt.Sprintf("visitedPeers_%s_%s.json", startStamp, endStamp))
	crawlLib.WritePeergraph(report, config.Outpath + fmt.Sprintf("peerGraph_%s_%s.csv", startStamp, endStamp))
	if config.UseCache{
		crawlLib.SaveNodeCache(report, config.CacheFile)
		log.WithField("File", config.CacheFile).Info("Online nodes saved in cache")
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
