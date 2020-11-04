#!/usr/bin/env bash

## master_script.sh: To be called from crontab.
## Invokes the crawler for 5 times and computes the eval
cd "$(dirname "$0")"

./crawl_times.sh 3

cd eval
echo "Number of nodes computation..."
Rscript num_nodes.R
echo "Agent version computation..."
Rscript agentVersion.R
echo "Generate number of nodes plot..."
Rscript plotting/num_nodes_plot.R
echo "Generate agent version plot..."
Rscript plotting/agent_version_plot.R
echo "Transforming plotdata to wide format..."
Rscript transform_narrow_num_nodes_to_broad.R
Rscript transform_narrow_av_to_broad.R
cd ..

mv data/*.json data/processed_data
mv data/*.csv data/processed_data

mv eval/figures/*.png /var/www/html/figs/
mv eval/*_broad.csv /var/www/html/data/

#sed -i -e "s/;/,/g" /var/www/html/data/num_nodes.csv
#sed -i -e "s/;/,/g" /var/www/html/data/agent_versions.csv
