# Tables created by the evaluation

All tables are saved in .tex format, so they can be easily included in a latex-document.

### Basic Statistics on the Degree Distributions

File ```degree_stats.tex```. Has the minimum, maximum, mean and median degree values over all crawls, distinguished by all nodes and only reachable ones.

### Protocol Statistics

File ```protocol_stats.tex```. For every protocol found in all multiaddresses during the crawl it depics the absolute number of peers supporting this protocol, as well as the relative percentage.
Note that if a peer has several, e.g., IPv4 addresses, the evaluation scripts count them only once as peer "supports IPv4".
Also note that this table is cumulative over all crawls, i.e., if a peer supported IPv6 during some crawl but not all of the time, we still count it as "supports IPv6".

### Uptime (=session lengths)

File ```session_lengths.tex```. Simply put, a table of uptime durations.
If we saw a peer in consecutive crawls, we make the (reasonable for back-to-back crawls) assumption that it was online between the two crawls as well.
This table summarizes the results: it shows the absolute and relative number of sessions that were longer than 5 minutes, 10 minutes, ..., 6 days.

### Geographical IP distribution

File ```geoIP_statistics.tex```. A table depicting the country distribution of peers averaged over all crawls. Only considers the top 10 countries by number of peers in them to avoid visual clutter.
Since the counts are averaged over all crawls, the confidence intervals are given as well.
Again, this is distinguished by all nodes and only nodes that were reachable by the crawler.
