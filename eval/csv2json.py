#!/usr/bin/env python3

import json
import os
import distutils.util
import sys


def readCSV(crawlDir, file):
	with open(crawlDir + file, "r") as f:
		lines = [l for l in f]
		return lines

def splitMultiAddrs(mastring):
	return mastring.strip("[]").split()

def splitLine(line):

	nodeDict = {}

	splitted = line.split(";")
	nodeid = splitted[0]
	multiaddrs = splitted[1]
	reachable = splitted[2]
	agentVersion = splitted[3]

	nodeDict["NodeID"] = nodeid
	nodeDict["MultiAddrs"] = splitMultiAddrs(multiaddrs)
	nodeDict["reachable"] = bool(distutils.util.strtobool(reachable))
	nodeDict["agent_version"] = agentVersion

	return nodeDict

def extractTimeStamp(fname):
	split = fname.split("_")
	start = split[1]
	end = split[2].strip(".csv")

	return((start, end))

if len(sys.argv) > 1:
		crawlDir = sys.argv[1]
else:
	crawlDir = "../output_data_crawls/"


visitedPeersFiles = [x for x in os.listdir(crawlDir) if x.startswith("visited") and x.endswith(".csv")]

for pf in visitedPeersFiles:
	print("Converting " + pf + "...")
	start, end = extractTimeStamp(pf)
	print(extractTimeStamp(pf))
	fileDict = {}
	fileDict["start_timestamp"] = start
	fileDict["end_timestamp"] = end

	content = readCSV(crawlDir, pf)
	fileDict["Nodes"] = [splitLine(line.strip("\n")) for line in content]

	jsonFileName = pf.split(".")[0] + ".json"
	with open(crawlDir + jsonFileName, "w") as outfile:
		json.dump(fileDict, outfile)