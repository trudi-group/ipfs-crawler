##### BASIC CRAWL INFORMATION #####

## Basic crawl-info like the length of the crawl etc.
crawlinfo:
	Rscript basic_crawl_information.R

##### PLOTS ######

### Number of nodes plot (Fig. 7 in the Paper)

number_of_nodes_plot: plot_data/num_nodes.csv figures/num_nodes.png
## First, pre-process the data
plot_data/num_nodes.csv:
	Rscript num_nodes.R

## Actual plotting

figures/num_nodes.png: plot_data/num_nodes.csv
	Rscript plotting/num_nodes_plot.R

### Specific degree distribution (Fig. 8 + Fig. 9 in the paper)

specific_degree_distribution: figures/log_all_nodes_degree_distribution.png

## As a command line argument, the RScript takes the index of the crawl whose degree distribution should be plotted.
## In this case we use the first (R starts enumerating at 1).
figures/log_all_nodes_degree_distribution.png:
	Rscript plotting/plot_specific_degree_distribution.R 1

### Agent version plot

agent_version_distribution: plot_data/agent_versions.csv figures/agent_version_distribution.png

plot_data/agent_versions.csv:
	Rscript agentVersion.R

figures/agent_version_distribution.png: plot_data/agent_versions.csv
	Rscript plotting/agent_version_plot.R

##### TABLES #####

### Session lengths (Tab. I in the paper). Session length = uptime

session_length_table: plot_data/session_lengths.csv tables/session_lengths.tex

## Preprocessing
plot_data/session_lengths.csv: 
	python3 sessionLengths.py

## Actual creation of table. We depend on crawlinfo, since we load some data about the crawl from the .lua key-value-store
tables/session_lengths.tex: plot_data/session_lengths.csv crawlinfo
	Rscript plotting/output_sessionLengths.R


### GeoIP distribution (Tab. II in the paper). This is a little bit involved:
## We use python and maxmind geo IP databases to resolve all the addresses. To have maximum flexibility on what statistics to compute,
## we simply duplicate the visitedPeers.csv files of the crawl, augmented with the geoIP lookup.
## The data added includes the country code, autonomous systems number and autonomous systems organization.
## Furthermore, for some reason, the lookup in Maxmind DBs takes forever, so this computation is quite lenghty.

geo_IP_distribution: maxmindDB geoIP_preprocessing plot_data/geoIP_per_crawl.csv tables/geoIP_statistics.tex

maxmindDB: geoipDBs/GeoLite2-Country.mmdb geoipDBs/GeoLite2-ASN.mmdb

geoipDBs/GeoLite2-Country.mmdb:
	$(error Please download the maxmind geo IP databases at https://dev.maxmind.com/geoip/geoip2/downloadable/. We need the GeoLite2-Country.mmdb and GeoLite2-ASN.mmdb.)

geoipDBs/GeoLite2-ASN.mmdb: geoipDBs/GeoLite2-Country.mmdb


geoIP_preprocessing:
	python3 geoIP_from_crawl.py

plot_data/geoIP_per_crawl.csv:
	Rscript geoIP.R

tables/geoIP_statistics.tex: plot_data/geoIP_per_crawl.csv
	Rscript plotting/geoIP_plot.R

### Protocol statistics (Tab. III in the paper).
protocol_statistics_table: plot_data/protocol_stats.csv  tables/protocol_stats.tex

## Preprocessing
plot_data/protocol_stats.csv:
	python3 extractProtocolStats.py

## Creation of table
tables/protocol_stats.tex: plot_data/protocol_stats.csv
	Rscript plotting/output_protocol_stats.R

### Degree Statistics Table (Tab. IV in the paper).
degree_statistics_table: plot_data/degree_stats.csv tables/degree_stats.tex

plot_data/degree_stats.csv:
	Rscript degree_stats.R

tables/degree_stats.tex: plot_data/degree_stats.csv
	Rscript plotting/output_degree_stats.R

##### REPORT #####

report: report.pdf

report.pdf:
	latexmk -lualatex -f report.tex
	mv report.pdf plot_data/report.pdf

##### META COMMANDS #####

plots: number_of_nodes_plot specific_degree_distribution agent_version_distribution

tabs: session_length_table protocol_statistics_table degree_statistics_table geo_IP_distribution

all: crawlinfo plots tabs report

## Delete the figures and tables
clean:
	rm -f figures/num_nodes.tex
	rm -f figures/num_nodes.png
	rm -f figures/log_online_nodes_degree_distribution.tex
	rm -f figures/log_all_nodes_degree_distribution.tex
	rm -f figures/log_all_nodes_degree_distribution.png
	rm -f figures/log_online_nodes_degree_distribution.png
	rm -f figures/agent_version_distribution.tex
	rm -f figures/agent_version_distribution.png

	rm -f tables/degree_stats.tex
	rm -f tables/protocol_stats.tex
	rm -f tables/session_lengths.tex
	rm -f tables/all_countries_geoIP_statistics.tex
	rm -f tables/geoIP_statistics.tex
	rm -f tables/full_agent_version_tab.tex
	rm -f report.pdf
	latexmk -c