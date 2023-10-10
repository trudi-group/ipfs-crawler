package crawling

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

// crawlOutputJSON is a helper struct to serialize the output of a crawl to
// JSON.
type crawlOutputJSON struct {
	StartDate time.Time         `json:"start_timestamp"`
	EndDate   time.Time         `json:"end_timestamp"`
	Nodes     []crawledNodeJSON `json:"found_nodes"`
}

// crawledNodeJSON is a helper struct to serialize the result of probing a
// single node to JSON.
// The fields ConnectionError and Result are mutually exclusive.
type crawledNodeJSON struct {
	ID         peer.ID        `json:"id"`
	MultiAddrs []ma.Multiaddr `json:"multiaddrs"`

	ConnectionError *string              `json:"connection_error"`
	Result          *crawledNodeDataJSON `json:"result"`
}

// crawledNodeDataJSON is a helper struct to serialize information about a
// single node to JSON.
// The field CrawlError indicates whether an error occurred during crawling.
type crawledNodeDataJSON struct {
	AgentVersion       string        `json:"agent_version"`
	SupportedProtocols []protocol.ID `json:"supported_protocols"`

	CrawlBeginTs time.Time `json:"crawl_begin_ts"`
	CrawlEndTs   time.Time `json:"crawl_end_ts"`
	CrawlError   *string   `json:"crawl_error"`

	PluginData map[string]pluginResultJSON `json:"plugin_data"`
}

// pluginResultJSON is a helper struct to serialize information about executing
// a plugin on a connectable node to JSON.
// The fields Error and Result are mutually exclusive.
type pluginResultJSON struct {
	BeginTimestamp time.Time   `json:"begin_timestamp"`
	EndTimestamp   time.Time   `json:"end_timestamp"`
	Error          *string     `json:"error"`
	Result         interface{} `json:"result"`
}

func (r nodeCrawlStatus) toCrawledNode(addrBook map[peer.ID][]ma.Multiaddr, id peer.ID) crawledNodeJSON {
	addr := addrBook[id]
	res := crawledNodeJSON{
		ID:         id,
		MultiAddrs: addr,
	}
	if r.err != nil {
		tmp := r.err.Error()
		res.ConnectionError = &tmp
		return res
	}

	res.Result = new(crawledNodeDataJSON)
	res.Result.AgentVersion = r.result.info.AgentVersion
	res.Result.SupportedProtocols = r.result.info.SupportedProtocols

	if len(r.result.pluginResults) != 0 {
		res.Result.PluginData = make(map[string]pluginResultJSON)

		for pn, pd := range r.result.pluginResults {
			tmp := pluginResultJSON{
				BeginTimestamp: pd.beginTimestamp,
				EndTimestamp:   pd.endTimestamp,
				Error:          nil,
				Result:         pd.result,
			}
			if pd.err != nil {
				tmp2 := pd.err.Error()
				tmp.Error = &tmp2
			}
			res.Result.PluginData[pn] = tmp
		}
	}

	res.Result.CrawlBeginTs = r.result.crawlDataBeginTs
	res.Result.CrawlEndTs = r.result.crawlDataEndTs
	if r.result.crawlDataError != nil {
		tmp := r.result.crawlDataError.Error()
		res.Result.CrawlError = &tmp
		return res
	}

	return res
}

// WriteMetadata writes a JSON report about the crawl to a file.
// The report contains metadata about each node.
func (report *CrawlOutput) WriteMetadata(startTs time.Time, endTs time.Time, path string) error {
	var nodes []crawledNodeJSON
	for id, node := range report.nodes {
		nodes = append(nodes, node.toCrawledNode(report.addrInfo, id))
	}
	crawlOutput := crawlOutputJSON{StartDate: startTs, EndDate: endTs, Nodes: nodes}

	// Open output file.
	vf, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}

	err = json.NewEncoder(vf).Encode(crawlOutput)
	if err != nil {
		return fmt.Errorf("unable to write output: %w", err)
	}

	return vf.Close()
}

// WritePeergraph writes the graph structure of the network as determined
// through the crawl to a CSV file.
func (report *CrawlOutput) WritePeergraph(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}

	w := csv.NewWriter(f)

	err = w.Write([]string{"source", "target", "target_crawlable", "source_crawl_timestamp"})
	if err != nil {
		return fmt.Errorf("unable to write output: %w", err)
	}
	for id, node := range report.nodes {
		if node.err != nil || node.result.crawlDataError != nil {
			continue
		}
		ts := node.result.crawlDataEndTs.Format(time.RFC3339)
		for _, neighbour := range node.result.crawlNeighbors {
			crawlable := fmt.Sprintf("%t", report.nodes[neighbour].err == nil && report.nodes[neighbour].result.crawlDataError == nil)
			err = w.Write([]string{id.String(), neighbour.String(), crawlable, ts})
			if err != nil {
				return fmt.Errorf("unable to write output: %w", err)
			}
		}
	}

	w.Flush()
	if err = w.Error(); err != nil {
		return fmt.Errorf("unable to flush CSV writer: %w", err)
	}

	return f.Close()
}

// RestoreNodeCache restores a list of peer addresses from a file.
func RestoreNodeCache(path string) ([]peer.AddrInfo, error) {
	nodedata, err := os.ReadFile(path)
	if err != nil {
		log.WithField("err", err).Warning("Node caching is enabled, but we couldn't read from the cache file. " +
			"Maybe this is the first run? Continuing without node cache this time.")
		return nil, fmt.Errorf("unable to read node cache: %w", err)
	}

	var result []peer.AddrInfo
	err = json.Unmarshal(nodedata, &result)
	if err != nil {
		return nil, fmt.Errorf("unable to decode node cache: %w", err)
	}

	return result, nil
}

// SaveNodeCache saves a list of peer addresses to file.
func (report *CrawlOutput) SaveNodeCache(cacheFile string) error {
	var nodesSave []peer.AddrInfo
	for id, node := range report.nodes {
		if node.err != nil || node.result.crawlDataError != nil {
			continue
		}
		nodesSave = append(nodesSave, peer.AddrInfo{
			ID:    id,
			Addrs: report.addrInfo[id],
		})
	}

	f, err := os.Create(cacheFile)
	if err != nil {
		return fmt.Errorf("unable to create node cache file: %w", err)
	}

	encoder := json.NewEncoder(f)
	err = encoder.Encode(nodesSave)
	if err != nil {
		return fmt.Errorf("unable to write node cache: %w", err)
	}

	return f.Close()
}
