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

##### META COMMANDS #####

plots: number_of_nodes_plot


all: crawlinfo plots

## Delete the figures and tables
clean:
	rm figures/num_nodes.tex
	rm figures/num_nodes.png