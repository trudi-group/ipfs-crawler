#!/usr/bin/env python3

# Time differences in final .csv file are in **Seconds**

import datetime
import sys
import os
from operator import itemgetter
import numpy as np
import time
import json

# Extracts the starting time of a crawl from its filename.
# Format of the crawlnames is, e.g., visitedPeers_30-11-19--23:54:17_30-11-19--23:58:24.csv
def extractTimestamp(filename):
	sname = filename.split("_")
	startTime = datetime.datetime.strptime(sname[1], "%d-%m-%y--%H:%M:%S")
	return startTime

# Returns the node ID from a line in the .csv
def extractIDFromLine(line, offlineEnabled):
	return(line.split(";")[0] if "true" in line.split(";")[2] or offlineEnabled else None)

def startSession(sdict, nodeid, time):
	tmpsessions = sdict.get(nodeid, [])
	tmpsessions.append((time, None))
	sdict[nodeid] = tmpsessions

def endSession(sdict, nodeid, time):
	# In this case we are sure that the nodeID is already in the dict. No need for a default value.
	tmpsessions = sdict[nodeid]
	tmpsessions[-1] = (tmpsessions[-1][0], time)
	sdict[nodeid] = tmpsessions


if __name__ == "__main__":

	if len(sys.argv) > 1:
		crawlDir = sys.argv[1]
	else:
		crawlDir = "../output_data_crawls/"

	dictOutFile = "plot_data/session_lengths.csv"
	rawDictOut = "rawDict"
	offlineEnabled = False

	# Sort the crawls by their startDate. To keep this ordering is crucical for the correctness of the session length calculation
	crawlFiles = sorted([(fname, extractTimestamp(fname)) for fname in os.listdir(crawlDir) if fname.startswith("visited")],\
		key=itemgetter(1))

	currentlyOnline = np.array([])
	# The dict has a list for every node ID that we encounter. The list is a list of tuples with (session start time, session end time)
	sessionDict = {}

	# ftuple is (filename, crawl time)
	# We track the sessions until the second to last file, the last file is then just used for "closing" the existing sessions
	for ftuple in crawlFiles[:-1]:
		## Crude progress indicator
		print(ftuple)
		with open(crawlDir+ftuple[0], "r") as f:
			crawldata = json.load(f)

			currentCrawlIDs = [node["NodeID"] for node in crawldata["Nodes"] if node["reachable"] or offlineEnabled]
			# Add all seen IDs (with their starting time) to the dict, if they are not present.
			# The idea is to keep track of the currently online IDs for each crawl and update the session dictionary
			# accordingly.

			# For each ID that was only but is not online now: remove from the set, add the end time of the session to the dict.
			# For each ID that is online now but is not in the currentlyOnline set, add it to the set and mark the start date
			# currentCrawlIDs = currentCrawlIDs[0:4]

			# Setdiff1d(a, b): return values in a but not in b.
			# We need these sets to update our session dictionary, and not to compute the set of currentlyOnline nodes.
			onlineButNotOnlineBefore = np.setdiff1d(currentCrawlIDs, currentlyOnline)
			offlineNow = np.setdiff1d(currentlyOnline, currentCrawlIDs)

			for node in onlineButNotOnlineBefore:
				startSession(sessionDict, node, ftuple[1])

			for node in offlineNow:
				endSession(sessionDict, node, ftuple[1])

			# Update the nodes that are "currently" online
			currentlyOnline = currentCrawlIDs
	
	# All crawl files have been processed, finish the remaining sessions
	for node in currentlyOnline:
		endSession(sessionDict, node, crawlFiles[-1][1])
	
	print("Writing dict to file...")
	## Write the dictionary to file
	with open(dictOutFile, "w") as f:
		for entry in sessionDict:
			for timeTuple in sessionDict[entry]:
				f.write("%s;%d\n" % (entry, (timeTuple[1]-timeTuple[0]).total_seconds()))

	# print("Writing dict with time tuples to file...")

	# with open(rawDictOut, "w") as f:
	# 	for entry in sessionDict:
	# 		f.write("%s;%s\n" % (entry, sessionDict[entry]))
