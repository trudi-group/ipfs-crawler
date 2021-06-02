package watcher

import(
    cid "github.com/ipfs/go-cid"
    log "github.com/sirupsen/logrus"
    _  "github.com/multiformats/go-multiaddr"
    _  "github.com/libp2p/go-libp2p-core/peer"
    "os"
    "strings"
    "fmt"
    "bufio"
    "encoding/json"
)

func ReadFile(filename string) []string {
    out := make([]string, 0)
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if !strings.HasPrefix(scanner.Text(), "#"){
            out = append(out, scanner.Text())
        }
    }
    return out
}

func ReadCids (filename string) []cid.Cid {
    data := ReadFile(filename)
    out := make([]cid.Cid, len(data))
    for i := range out {
        out[i],_ = cid.Decode(data[i])
    }
    return out
}



func ToJson(data []*Event, filename string){
    //unifiy with Peer as key

    f, err := os.Create(filename)
    if err != nil {
        fmt.Println(err)
        return
    }
    defer f.Close()
    jsonite := json.NewEncoder(f)
    // for _, slice := range data {
    //         jsonite.Encode(slice)
    // }
    jsonite.Encode(data)
}
