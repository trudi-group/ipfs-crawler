package crawling
import(
  peer "github.com/libp2p/go-libp2p-core/peer"
  ma "github.com/multiformats/go-multiaddr"
  "encoding/json"
  "os"
  log "github.com/sirupsen/logrus"
  "fmt"
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
  fmt.Fprintf(f, "SOURCE;TARGET;ONLINE\n")
  for _, node := range report.Nodes {
    for _, neigh := range node.Neighbours{
      on := report.Nodes[neigh].Reachable
      fmt.Fprintf(f, "%s;%s;%t\n", node.NID, neigh, on)
    }

  }
}
