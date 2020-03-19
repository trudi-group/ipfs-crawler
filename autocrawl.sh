#!/usr/bin/env bash

# USAGE: ./autocrawl.sh [-l] <crawl duration in days>
# Duration is given in days, convert to seconds
usage="$0 [-l <dir>] <crawl duration in days>\n
		-l: log directory *that has to exist*. If not specified, logs are written to /dev/null"

logdir=""

case $1 in
	-l )	shift
	logdir=$1
	if [[ ! -d $logdir ]]; then
		echo "$dir is not a directory, will not keep logs there."
		exit 1
	fi
	echo "Outputting logs to $logdir."
	shift
		;;
esac

if [[ "$logdir" == "" ]]; then
	echo "Not keeping logs."
fi

if [[ "$1" == "" ]]; then
    echo "No duration given."
    echo -e $usage
    exit 1
fi

if ! [[ "$1" =~ ^[0-9]+$ ]]
    then
        echo "Duration must be integer"
        exit 1
fi

duration=$1

secondDuration=$duration*24*3600
startTime=$(date +%s)
endTime=$((startTime+secondDuration))

counter=1

echo -e "Started crawling at $(date --date=@+$startTime).\nWill crawl until $(date --date=@+$endTime)."

while [[ $(date +%s) -le $endTime ]]
do
	echo "Crawl no. $counter"
	if [[ "$logdir" == "" ]]; then
		./start_crawl 2> /dev/null
	else
		./start_crawl 2> $logdir/crawl_log_"$(date --rfc-3339='seconds')"_$counter
	fi
	((counter++))
done