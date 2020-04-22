#!/usr/bin/env bash

## master_script.sh: To be called from crontab.
## Invokes the crawler for 5 times and computes the eval

./crawl_times 5

cd eval
echo "Number of nodes computation..."
Rscript num_nodes.R
echo "Agent version computation..."
Rscript agentVersion.R
echo "Generate number of nodes plot..."
Rscript plotting/num_nodes_plot.R
echo "Generate agent version plot..."
Rscript plotting/agent_version_plot.R
cd ..

mv data/*.csv data/processed_data

mv figures/*.png /var/www/html/figs/