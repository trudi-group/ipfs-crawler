# A Crawler for the Kademlia-part of the IPFS-network

**For more details, see [our paper](https://arxiv.org/abs/2002.07747).**

If you use our work, please **cite our papers**:

Sebastian A. Henningsen, Martin Florian, Sebastian Rust, Björn Scheuermann:
**Mapping the Interplanetary Filesystem.** *Networking 2020*: 289-297\
[[BibTex]](https://dblp.uni-trier.de/rec/conf/networking/HenningsenFR020.html?view=bibtex)

Sebastian A. Henningsen, Sebastian Rust, Martin Florian, Björn Scheuermann:
**Crawling the IPFS Network.** *Networking 2020*: 679-680\
[[BibTex]](https://dblp.uni-trier.de/rec/conf/networking/HenningsenRF020.html?view=bibtex)

For a Live Version of the crawler results, check out our [Periodic Measurements of the IPFS Network](https://trudi.weizenbaum-institut.de/ipfs_analysis.html)

## In a Nutshell

This crawler is designed to enumerate all reachable nodes within the DHT/KAD-part of the IPFS network and return their neighborhood graph.
For each node it saves
* The ID
* All known multiaddresses that were found in the DHT
* Whether it was reachable by the crawler or not, i.e., if a connection attempt was successful
* The agent version.

This is achieved by sending multiple `FindNode`-requests to each node in the network, targeted in such a way that each request extracts the contents of exactly one DHT bucket.

The crawler is optimized for speed, to generate as accurate snapshots as possible.
It starts from the (configurable) bootstrap nodes, polls their buckets and continues to connect to every peer it has not seen so far.

For an in-depth dive and discussion to the crawler and the obtained results, you can watch @scriptkitty's talk at ProtocolLabs:

[![Link to YouTube](https://img.youtube.com/vi/jQI37Y25jwk/1.jpg)](https://www.youtube.com/watch?v=jQI37Y25jwk)

## Run one or multiple crawls

To run a single crawl simply do:

```bash
make build
./start_crawl
```

**Important note:** We ship the pre-images necessary for a successful crawl, but you can compute them yourself with `make preimages`.
Note that the preimages only have to be computed *once*, it'll take some minutes, to compute them, though.
`start_crawl` will be created at build time.

One crawl will take 5-10 minutes, depending on your machine.

For multiple crawls, use the `autocrawl.sh` script instead of `start_crawl` in the last line. It takes a duration in days and an optional directory to put logs into.
Note that there will be a lot of output on your disk, one week of crawling (without logs) can lead to 30-50GB of data!
The complete workflow is:

```bash
make build
./autocrawl [-l logdir] <crawl duration in days>
```

## Configuration

The crawler is configured via a YAML configuration file.
An example config with sane defaults is provided in [dist/](dist).

## Evaluation of Results

After running a few crawls, the output directory should have some data in it.
To run the evaluation and generate the same plots/tables as in the paper (and more!) you have the option to run it via Docker or manually.
We've compiled the details [in the README](./eval/README.md)

## Features

### Node Caching

If configured, the crawler will cache the nodes it has seen.
The next crawl will then not only start at the boot nodes but also add all previously reachable nodes to the crawl queue.
This can increase the crawl speed, and therefore the accuracy of the snapshots, significantly.
Due to node churn, this setting is most reasonable when performing many consecutive crawls.

### Sanity Check ("Canary Peers")

The crawler enumerates the nodes in the network, but without ground truth it is hard to assess the quality and completeness of a crawl.
Therefore, it might be desirable to check whether some known IPFS-nodes appear in the crawl.
This functionality used to exist in the crawler, but we believe it is more convenient to check externally using common UNIX tools.

## Output of a crawl

A crawl writes two files to the output directory configured via the configuration file:
* ```visitedPeers_<start_of_crawl_datetime>.json```
* ```peerGraph_<start_of_crawl_datetime>.csv```

### Format of ```visitedPeers```

```visitedPeers``` contains a json structure with meta information about the crawl as well as each found node.
Each node entry corresponds to exactly one node on the network and has the following fields:
```json
{
  "id": "<multihash of the node id>",
  "multiaddrs": ["<multiaddress_1>", "<multiaddress_2>", "<multiaddress_n>"],
  "crawlable": "<whether our crawler could connect to the peer and enumerate all its neighbors>",
  "agent_version": "<if a connection was successful, the agent version string>",
  "supported_protcols": "<if a connection was successful, a list of protocols identifiers supported by the peer>",
  "crawl_started_timestamp": "<timestamp of the first connection attempt to the peer>", 
  "crawl_finished_timestamp": "<timestamp after all interactions with the peer finished, whether successful or not>"
}
```

The Node's ID is a [multihash](https://github.com/multiformats/multihash), the addresses a peer advertises are [multiaddresses](https://github.com/multiformats/multiaddr).
```crawlable``` is true/false and indicates, whether the respective node could be reached by the crawler or not. Note that the crawler will try to connect to *all* multiaddresses that it found in the DHT for a given peer.
```agent_version``` is simply the agent version string the peer provides when connecting to it.

Data example (somewhat anonymized):
```json
{
  "multiaddrs": [
    "/ip4/127.0.0.1/udp/4001/quic",
    "/ip4/172.17.0.5/tcp/4001",
    "/ip4/54.37.x.x/tcp/50027",
    "/ip4/54.37.x.x/udp/4001/quic",
    "/ip4/127.0.0.1/tcp/4001",
    "/ip4/172.17.0.5/udp/4001/quic",
    "/ip4/54.37.x.x/tcp/4001",
    "/ip4/54.37.x.x/udp/4001/quic-v1"
  ],
  "agent_version": "kubo/0.19.0-dev/f73cd19/docker",
  "id": "QmZuP2QQeHGZ...",
  "crawlable": true,
  "crawl_started_timestamp": "2023-04-14T03:17:41.870584726+01:00",
  "crawl_finished_timestamp": "2023-04-14T03:17:42.105222692+01:00",
  "supported_protocols": [
    "/ipfs/bitswap/1.2.0",
    "/ipfs/bitswap/1.1.0",
    "/ipfs/bitswap",
    "/libp2p/circuit/relay/0.1.0",
    "/libp2p/circuit/relay/0.2.0/stop",
    "/ipfs/id/push/1.0.0",
    "/ipfs/ping/1.0.0",
    "/libp2p/autonat/1.0.0",
    "/libp2p/circuit/relay/0.2.0/hop",
    "/libp2p/dcutr",
    "/ipfs/lan/kad/1.0.0",
    "/ipfs/bitswap/1.0.0",
    "/x/",
    "/ipfs/kad/1.0.0",
    "/ipfs/id/1.0.0"
  ]
}
```

### Format of `peerGraph`

`peerGraph` is an edgelist, where each line in the file corresponds to one edge. A line has the form

```csv
source,target,target_crawlable,source_crawl_timestamp
```

Two nodes are connected, if the crawler found the peer `target` in the buckets of peer `source`.
Example line (somewhat anonymized):

```csv
12D3KooWD9QV2...,12D3KooWCDx5k1...,true,2023-04-14T03:18:06+01:00
```

which says that the peer with ID `12D3KooWD9QV2...` had an entry for peer `12D3KooWCDx5k1...` in its buckets and that the latter was reachable by our crawler.

If `target_crawlable` is `false`, this indicates that the crawler was not able to connect to or enumerate all of `target`'s peers.
Since some nodes reside behind NATs or are otherwise uncooperative, this is not uncommon to see.

## Bootstrap Peers

The crawler needs to know which peers to use to start a crawl.
These are configured via the configuration file.
To get the default bootstrap peers of an IPFS node, simply run ```./ipfs bootstrap list > bootstrappeers.txt```.

## Libp2p complains about key lengths

Libp2p uses minimum keylenghts of [2048 bit](https://github.com/libp2p/go-libp2p-core/blob/master/crypto/rsa_common.go), whereas IPFS uses [512 bit](https://github.com/ipfs/infra/issues/378).
Therefore, the crawler can only connect to one IPFS bootstrap node and refuses a connection with the others, due to this key length mismatch.
Libp2p can be configured to ignore this mismatch via an environment variable:

```bash
export LIBP2P_ALLOW_WEAK_RSA_KEYS=""
```

## Socket limit

ipfs-crawler uses a lot of sockets.
On linux, this can result into "too many sockets" errors during connections.
Please raise the maximum number of sockets on linux via 
```bash
ulimit -n unlimited
```
or equivalent commands on different platforms
