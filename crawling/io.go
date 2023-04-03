package crawling

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type CrawlOutputJSON struct {
	StartDate string             `json:"start_timestamp"`
	EndDate   string             `json:"end_timestamp"`
	Nodes     []*CrawledNodeJSON `json:"found_nodes"`
}
type CrawledNodeJSON struct {
	NID          peer.ID        `json:"NodeID"`
	MultiAddrs   []ma.Multiaddr `json:"multiaddrs"`
	Reachable    bool           `json:"reachable"`
	AgentVersion string         `json:"agent_version"`
}

func ReportToFile(report *CrawlOutput, path string) error {
	var nodes []*CrawledNodeJSON
	for _, node := range report.Nodes {
		jsonFormatted := CrawledNodeJSON{NID: node.NID, MultiAddrs: node.MultiAddrs, Reachable: node.Reachable, AgentVersion: node.AgentVersion}
		nodes = append(nodes, &jsonFormatted)
	}
	crawlOutput := CrawlOutputJSON{StartDate: report.StartDate, EndDate: report.EndDate, Nodes: nodes}

	// Open output file.
	vf, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}

	err = json.NewEncoder(vf).Encode(crawlOutput)
	if err != nil {
		return fmt.Errorf("unable to write output: %w", err)
	}

	return nil
}

func WritePeergraph(report *CrawlOutput, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}

	_, err = fmt.Fprintf(f, "SOURCE;TARGET;ONLINE;TIMESTAMP\n")
	if err != nil {
		return fmt.Errorf("unable to write output: %w", err)
	}
	for _, node := range report.Nodes {
		for _, neigh := range node.Neighbours {
			on := report.Nodes[neigh].Reachable
			time := node.Timestamp
			_, err = fmt.Fprintf(f, "%s;%s;%t;%s\n", node.NID, neigh, on, time)
			if err != nil {
				return fmt.Errorf("unable to write output: %w", err)
			}
		}
	}

	return nil
}

// RestoreNodeCache restores a viously cached file of nodes.
func RestoreNodeCache(path string) ([]*peer.AddrInfo, error) {
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

	var out []*peer.AddrInfo
	// switch to pointers to fullfil requirements of main.go... because this is stupid
	for it := range result {
		out = append(out, &result[it])
	}
	return out, nil
}

func SaveNodeCache(result *CrawlOutput, cacheFile string) error {
	var nodesSave []peer.AddrInfo
	for _, node := range result.Nodes {
		if node.Reachable {
			recreated := peer.AddrInfo{
				ID:    node.NID,
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

	return nil
}
