package crawling

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type crawlOutputJSON struct {
	StartDate time.Time         `json:"start_timestamp"`
	EndDate   time.Time         `json:"end_timestamp"`
	Nodes     []crawledNodeJSON `json:"found_nodes"`
}

type crawledNodeJSON struct {
	MultiAddrs             []ma.Multiaddr `json:"multiaddrs"`
	AgentVersion           string         `json:"agent_version"`
	ID                     peer.ID        `json:"id"`
	Crawlable              bool           `json:"crawlable"`
	CrawlStartedTimestamp  time.Time      `json:"crawl_started_timestamp"`
	CrawlFinishedTimestamp time.Time      `json:"crawl_finished_timestamp"`
	SupportedProtocols     []string       `json:"supported_protocols"`
}

func ReportToFile(report *CrawlOutput, startTs time.Time, endTs time.Time, path string) error {
	var nodes []crawledNodeJSON
	for _, node := range report.Nodes {
		jsonFormatted := crawledNodeJSON{
			MultiAddrs:             node.MultiAddrs,
			AgentVersion:           node.AgentVersion,
			ID:                     node.ID,
			Crawlable:              node.Crawlable,
			CrawlStartedTimestamp:  node.CrawlStartedTimestamp,
			CrawlFinishedTimestamp: node.CrawlFinishedTimestamp,
			SupportedProtocols:     node.SupportedProtocols,
		}
		nodes = append(nodes, jsonFormatted)
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

func WritePeergraph(report *CrawlOutput, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}

	w := csv.NewWriter(f)

	err = w.Write([]string{"source", "target", "target_crawlable", "source_crawl_timestamp"})
	if err != nil {
		return fmt.Errorf("unable to write output: %w", err)
	}
	for _, node := range report.Nodes {
		for _, neighbour := range node.Neighbours {
			crawlable := fmt.Sprintf("%t", report.Nodes[neighbour].Crawlable)
			ts := node.CrawlFinishedTimestamp.Format(time.RFC3339)
			err = w.Write([]string{node.ID.String(), neighbour.String(), crawlable, ts})
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

// RestoreNodeCache restores a viously cached file of nodes.
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

func SaveNodeCache(result *CrawlOutput, cacheFile string) error {
	var nodesSave []peer.AddrInfo
	for _, node := range result.Nodes {
		if node.Crawlable {
			recreated := peer.AddrInfo{
				ID:    node.ID,
				Addrs: node.MultiAddrs,
			}
			nodesSave = append(nodesSave, recreated)
		}
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
