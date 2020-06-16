#!/usr/bin/env python

import os
import sys
import json

def ExtractProtocols(maString):
	# The format of such an address is
	# /ip4|ip6/<ip>/tcp|udp/<port>[/quic]
	# Example: /ip4/147.75.199.141/tcp/24697
	return maString.split("/")[1]

# Counts the seen protocols, though it counts each occurrence only once
# (no need to overcount ipv4 if a node has a trillion v4 addresses, e.g.)
def updateProtocolDict(protDict, protList, nodeid, timestamp):
	## This code is for extracting the protocols per crawl for each peer
	# peerProtocols = protDict.get(nodeid, [])

	# prots = list(set(protList))
	# peerProtocols.append((timestamp, prots))
	# protDict[nodeid] = peerProtocols

	## This code merely aggregates the protocols for each peer
	peerProtocols = set(protocolDict.get(nodeid, []))
	prots = list(set(protList).union(peerProtocols))
	protDict[nodeid] = list(prots)

def outputProtocolDict(protDict, path):
	# Erase existing files ("w" argument)
	with open(path, "w") as f:
		f.write("nodeid;protocol\n")
		for k, v in protDict.items():
			protocols = getUniqueProts(v)
			for p in protocols:
				f.write(k + ";" + p + "\n")

def getUniqueProts(dictEntry):
	## The dict entries are [(<ts>, [<p1, p2, ...>]), (.,.), ...]
	prots = []
	## For aggregated version, dict entries are just []
	for e in dictEntry:
		prots += [e]
	return set(prots)

def extractTimestamp(filename):
	# Example for a filename: visitedPeers_29-11-19--10:30:25_29-11-19--10:41:29.csv
	return filename.split("_")[1]

############# MAIN ############

if __name__ == "__main__":
	if len(sys.argv) > 1:
		crawlDir = sys.argv[1]
	else:
		crawlDir = "../output_data_crawls/"

	outProtPath = "./plot_data/protocol_stats.csv"
	
	allFiles = os.listdir(crawlDir)
	peerFiles = [x for x in allFiles if x.startswith("visited")]
	
	onlyOnlineNodes = False

	protocolDict = {}

	i = 0
	for crawlFile in peerFiles:
		i += 1
		ts = extractTimestamp(crawlFile)
		print(crawlFile)
		print("Progress: " + str(i/len(peerFiles)))
		with open(crawlDir+crawlFile, "r") as f:
			crawldata = json.load(f)

			for node in crawldata["Nodes"]:
				nodeid = node["NodeID"]
				rawMA = node["MultiAddrs"]
				# We had at least one occurence of an ID without address -> ignore that
				if len(rawMA) == 0:
					continue
				MAProtArray = [ExtractProtocols(ma) for ma in rawMA]
				# We get (protocol, addr) back, if the address is ipv4 or ipv6
				updateProtocolDict(protocolDict, MAProtArray, nodeid, ts)
				

	outputProtocolDict(protocolDict, outProtPath)