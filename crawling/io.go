package crawling
import(
  peer "github.com/libp2p/go-libp2p-core/peer"
  ma "github.com/multiformats/go-multiaddr"
  "encoding/json"
  "os"
  log "github.com/sirupsen/logrus"
  "fmt"
  "io/ioutil"
)
type CrawlOutputJSON struct {
  StartDate string `json:"start_timestamp"`
  EndDate string `json:"end_timestamp"`
  Nodes []*CrawledNodeJSON `json:"found_nodes`
}
type CrawledNodeJSON struct {
  NID peer.ID `json:"NodeID"`
  MultiAddrs[] ma.Multiaddr `json:multiaddrs`
  Reachable bool `json:"reachable"`
  AgentVersion string `json:"agent_version"`
}

func ReportToFile(report *CrawlOutput, path string)  {

  var nodes []*CrawledNodeJSON
  for _, node := range report.Nodes {
    jsonFormatted := CrawledNodeJSON{NID: node.NID, MultiAddrs: node.MultiAddrs, Reachable: node.Reachable, AgentVersion: node.AgentVersion}
    nodes = append(nodes, &jsonFormatted)
  }

  crawlOutput := CrawlOutputJSON{StartDate: report.StartDate, EndDate: report.EndDate, Nodes: nodes}

  vf, err := os.OpenFile(path, os.O_CREATE | os.O_RDWR, 0666)
  if os.IsNotExist(err) {
    os.Mkdir(path, 0777)
  } else if err != nil {
    panic(err)
  }

  err = json.NewEncoder(vf).Encode(crawlOutput)
  if err != nil {
    log.WithField("err", err).Error("Could not encode JSON and/or write to output file.")
  }
}

func WritePeergraph(report *CrawlOutput, path string)  {
  f, err := os.OpenFile(path, os.O_CREATE | os.O_RDWR, 0666)
  if os.IsNotExist(err) {
    os.Mkdir(path, 0777)
  } else if err != nil {
    panic(err)
  }
  fmt.Fprintf(f, "SOURCE;TARGET;ONLINE;TIMESTAMP\n")
  for _, node := range report.Nodes {
    for _, neigh := range node.Neighbours{
      on := report.Nodes[neigh].Reachable
      time := node.Timestamp
      fmt.Fprintf(f, "%s;%s;%t;%s\n", node.NID, neigh, on, time)
    }

  }
}

// RestoreNodeCache restores a previously cached file of nodes.
func RestoreNodeCache(path string) ([]*peer.AddrInfo, error)  {
    nodedata, err := ioutil.ReadFile(path)
    if err != nil {
        log.WithField("err", err).Warning("Node caching is enabled, but we couldn't read from the cache file. " +
        	"Maybe this is the first run? Continuing without node cache this time.")
        return nil, err
    }
    var result []peer.AddrInfo
    err = json.Unmarshal(nodedata, &result)
    if err != nil {
        log.WithField("err", err).Error("Error unmarshalling from cacheFile.")
        return nil, err
    }
    var out []*peer.AddrInfo
    // switch to pointers to fullfil requirements of main.go... because this is stupid
    for _, val := range result{
        out = append(out, &val)
    }
    return out, nil
}
func SaveNodeCache(result *CrawlOutput, cacheFile string)  {
    nodesSave := []peer.AddrInfo{}
    for _, node :=  range result.Nodes {
        if node.Reachable {
            recreated := peer.AddrInfo{
                ID: node.NID,
                Addrs: node.MultiAddrs,
            }
            nodesSave = append(nodesSave, recreated)
        }
    }
    marshalled, _ := json.Marshal(nodesSave)
    err := ioutil.WriteFile(cacheFile, marshalled, 0644)
    if err != nil {
        log.WithField("err", err).Error("Error writting to cacheFile.")
        return
    }

}
