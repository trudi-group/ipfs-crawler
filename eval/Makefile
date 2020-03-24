##### BASIC CRAWL INFORMATION #####

## Basic crawl-info like the length of the crawl etc.
crawlinfo:
	Rscript basic_crawl_information.R

##### PLOTS ######

### Number of nodes plot (Fig. 7 in the Paper)

number_of_nodes_plot: plot_data/num_nodes.csv figures/num_nodes.tex
## First, pre-process the data
plot_data/num_nodes.csv:
	Rscript num_nodes.R

## Actual plotting

figures/num_nodes.tex: plot_data/num_nodes.csv
	Rscript plotting/num_nodes_plot.R

### Specific degree distribution (Fig. 8 + Fig. 9 in the paper)

specific_degree_distribution: figures/log_all_nodes_degree_distribution.tex

## As a command line argument, the RScript takes the index of the crawl whose degree distribution should be plotted.
## In this case we use the first (R starts enumerating at 1).
figures/log_all_nodes_degree_distribution.tex:
	Rscript plotting/plot_specific_degree_distribution.R 1

##### TABLES #####

### Session lengths (Tab. I in the paper). Session length = uptime

session_length_table: plot_data/session_lengths.csv tables/session_lengths.tex

## Preprocessing
plot_data/session_lengths.csv: 
	python sessionLengths.py

## Actual creation of table. We depend on crawlinfo, since we load some data about the crawl from the .lua key-value-store
tables/session_lengths.tex: plot_data/session_lengths.csv crawlinfo
	Rscript plotting/output_sessionLengths.R

### Protocol statistics (Tab. III in the paper).
protocol_statistics_table: plot_data/protocol_stats.csv  tables/protocol_stats.tex

## Preprocessing
plot_data/protocol_stats.csv:
	python extractProtocolStats.py

## Creation of table
tables/protocol_stats.tex: plot_data/protocol_stats.csv
	Rscript plotting/output_protocol_stats.R

##### META COMMANDS #####

plots: number_of_nodes_plot

tabs: session_length_table

all: crawlinfo plots tabs

## Delete the figures and tables
clean:
	rm figures/num_nodes.tex
	rm figures/num_nodes.png
	rm figures/log_online_nodes_degree_distribution.tex
	rm figures/log_all_nodes_degree_distribution.tex
	rm figures/log_all_nodes_degree_distribution.png
	rm figures/log_online_nodes_degree_distribution.png