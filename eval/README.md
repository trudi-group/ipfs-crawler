# Evaluation: Create Statistics Based on Obtained Crawl Data

Assume you have some ```.csv``` files from running a few crawls. In here are some tools to generated a .pdf with a lot of statistics on the crawl.

**Note that the evaluation may take several hours up to even a day, depending on how much crawl data you have!**

## Step-By-Step Guide Using Docker

1. Check that you're in the ```ipfs-crawler/eval/``` directory and build the image with ```docker build -t scriptkitty/ipfs-crawl-eval .```.
This takes some time and requires 4GB of space, since we use texlive-full to build a PDF in the end -- any contribution towards making this image smaller is appreciated, since it's not on our priority list.

2. After the build, go to the parent directory (```ipfs-crawler/```) and run the container with the provided script ```./run_docker_eval.sh```.
The script will automatically use the data in ```ipfs-crawler/output_data_crawls```, if you want to provide a custom data folder, just provide the **absolute** path:
```./run_docker_eval.sh /path/to/crawl/data```

Especially the geoIP lookup can take up to 1-2 days, so this is best run on a dedicated server. 
The run will populate the directories on the host and output a ```report.pdf``` in the end, containing the computed statistics, tables and plots.

For details on the generated files, see the descriptions in the READMEs: [```figures```](https://github.com/scriptkitty/ipfs-crawler/blob/master/eval/figures/README.md), [```tables```](https://github.com/scriptkitty/ipfs-crawler/blob/master/eval/tables/README.md).

## Running it on Your Own Maschine

### Install Dependencies

In Ubuntu, these are the necessary packages:

	r-base
	python3
	texlive-full
	latexmk

```r-base``` and ```python3``` for computing statistics and plotting, ```telive-full``` and ```latexmk``` to build a report in the end.
#### R packages

	Rscript -e "install.packages(c(\"data.table\", \"reshape2\", \"ggplot2\", \"scales\", \
             \"tikzDevice\", \"stringr\", \"pbapply\", \"igraph\", "jsonlite", "tidyr"))"

#### Python3 packages

	pip3 install geoip2 numpy ip2location

#### Run the evaluation

The evaluation consists of several basic statistics, figures, tables and the report in the end. To build everything, simply issue
	
	make all

which will output a ```report.pdf``` in the ```eval/``` directory.

Figures are also outputted as ```.png```s to the ```figures/``` directory, to build only them, for example, use

	make plots
