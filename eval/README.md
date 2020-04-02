# Evaluation: Create Statistics Based on Obtained Crawl Data

Assume you have some ```.csv``` files from running a few crawls. In here are some tools to generated a .pdf with a lot of statistics on the crawl.

**Note that the evaluation may take several hours, depending on how much crawl data you have!**

## Dependencies

	r-base
	python3
	texlive-full
	latexmk

```r-base``` and ```python3``` for computing statistics and plotting, ```telive-full``` and ```latexmk``` to build a report in the end.
### R packages

	Rscript -e "install.packages(c(\"data.table\", \"reshape2\", \"ggplot2\", \"scales\", \
             \"tikzDevice\", \"stringr\", \"pbapply\", \"igraph\"))"

### Python3 packages

	pip install geoip2 numpy

## Run the evaluation

The evaluation consists of several basic statistics, figures, tables and the report in the end. To build everything, simply issue
	
	make all

which will output a ```report.pdf``` in the ```eval/``` directory.

Figures are also outputted as ```.png```s to the ```figures/``` directory, to build only them, for example, use

	make figures
