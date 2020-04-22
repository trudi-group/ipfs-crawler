#!/usr/bin/env bash

# USAGE: Crawl for a certain number of times:
# ./crawl_times n

counter=1

if [[ "$1" == "" ]]; then
    echo "No n given."
    exit 1
fi

times=$1
cd ../

while [[ $counter -le $times ]]
do
	echo "Crawl no. $counter"
	./start_crawl #2> /dev/null
	((counter++))
done