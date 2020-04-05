#!/usr/bin/env bash

DEFAULTDIR="$(pwd)/output_data_crawls"
DATADIR=""
uid=$(id -u $USER)

if [[ $1 == "" ]]; then
		DATADIR=$DEFAULTDIR
		echo "No data directory provided, using default $DATADIR"
else
	DATADIR=$1
fi

if [[ ! "$DATADIR" == /* ]]; then
	echo "Docker needs absolute paths!"
	exit 1
fi

echo "Running eval-docker with data directory $DATADIR..."
sudo docker run \
	--mount type=bind,source=$DATADIR,target=/output_data_crawls \
	--mount type=bind,source=$(pwd)/eval/plot_data,target=/eval/plot_data \
	--mount type=bind,source=$(pwd)/eval/figures,target=/eval/figures \
	--mount type=bind,source=$(pwd)/eval/tables,target=/eval/tables \
	scriptkitty/ipfs-crawl-eval

echo "Finished eval, copying report.pdf to $(pwd):"
cp plot_data/report.pdf $(pwd)/report.pdf