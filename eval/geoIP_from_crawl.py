#!/usr/bin/env python

import os
import sys
import geoip2.database
import ipaddress
import tempfile

from maxminddb import (MODE_AUTO, MODE_MMAP, MODE_MMAP_EXT, MODE_FILE,
                       MODE_MEMORY, MODE_FD)


def ParseMA(maString):
	# The format of such an address is
	# /ip4|ip6/<ip>/tcp|udp/<port>[/quic]
	# Example: /ip4/147.75.199.141/tcp/24697
	# Possible ToDo: right now we only consider ipv4 and ipv6 addrs. Those are the majority of all addresses anyway, but
	# maybe include, e.g., dns4 and dns6 in the future.
	addrsplit = maString.split("/")
	protocol = addrsplit[1]
	addr = None
	# This yields a '' entry (left of the first / is nothing)
	if "ip4" in protocol:
		addr = addrsplit[2]

	elif "ip6" in protocol:
		addr = addrsplit[2]

	return (protocol, addr)


def extractTimestamp(filename):
	# Example for a filename: visitedPeers_29-11-19--10:30:25_29-11-19--10:41:29.csv
	return filename.split("_")[1]

############# MAIN ############

if __name__ == "__main__":
	if len(sys.argv) > 1:
		crawlDir = sys.argv[1]
	else:
		crawlDir = "../output_data_crawls/"

	outputDir = "../output_data_crawls/geoIP_processing/"

	if not os.path.exists(outputDir):
		os.makedirs(outputDir)

	allFiles = os.listdir(crawlDir)
	peerFiles = [x for x in allFiles if x.startswith("visited")]
	processedFiles = os.listdir(outputDir)
	
	onlyOnlineNodes = False
	geoIPPathCountry = "geoipDBs/GeoLite2-Country.mmdb"
	geoIPPathAS = "geoipDBs/GeoLite2-ASN.mmdb"
	geoIPDB = geoip2.database.Reader(geoIPPathCountry)#, mode=MODE_MMAP_EXT)
	ASDB = geoip2.database.Reader(geoIPPathAS)#, mode=MODE_MMAP_EXT)

	i = 0
	for crawlFile in peerFiles:
		if crawlFile in processedFiles:
			print("Already processed " + crawlFile + ", skipping file...")
			continue

		ts = extractTimestamp(crawlFile)
		multiCountry = 0
		print(crawlFile)
		print("Progress: " + str(i*100/len(peerFiles)) + "%")
		totalCountryCount = 0
		tmpf = tempfile.NamedTemporaryFile(mode="r+")
		tmpf.write("nodeid;IP;country;ASNO;ASName;online;agentVersion\n")
		with open(crawlDir+crawlFile, "r") as f:
			lines = [l for l in f]

			for l in lines:
				# Roadmap: 
				# * Take each line and split it into its three components
				# * Lookup the country code, AS-no and AS name for each IP
				# * Put everything back together
				# * Write back to *the same* file (through a temp-file)
				s = l.split(";")
				nodeid = s[0]
				nodeOnline = s[2]
				agentVersion = s[3]
				# Strip the brackets around the MA list, split at the whitespaces, extract the IP (if possible) and ignore 'None' addresses
				rawMA = s[1].strip("[]")
				# We had at least one occurence of an ID without address -> ignore that
				if len(rawMA) == 0:
					print("No addresses to peer ID " + nodeid)
					continue

				rawMA = rawMA.split(" ")
				MAProtArray = [ParseMA(ma) for ma in rawMA]
				# We get (protocol, addr) back, if the address is ipv4 or ipv6
				AddrArray = [ma[1] for ma in MAProtArray if ma[1]]
				# out format for the output is nodeID;[<ip, country_code, as_no, as_name>, <...>, ...];true/false
				for ip in AddrArray:
					if ipaddress.ip_address(ip).is_private or ipaddress.ip_address(ip).is_link_local:
						addrs = str(ip) + ";LocalIP;NA;NA"
					else:
						try:
							countryc = geoIPDB.country(ip).country.iso_code
							if countryc == None:
								countryc = "Unknown"

							resp = ASDB.asn(ip)
							addrs = str(ip) + ";" + str(countryc) + ";" + str(resp.autonomous_system_number) +\
							";" + str(resp.autonomous_system_organization)

						except geoip2.errors.AddressNotFoundError:
							# Just append "Unknown"
							addrs = str(ip) + ";Unknown;NA;NA"
					# This code assumes the updated output format with 4 columns, the 4th being the agent version
					tmpf.write(nodeid + ";" + addrs + ";" + nodeOnline + ";" + agentVersion)

		tmpf.seek(0)
		with open(outputDir+crawlFile, "w") as f:
			for l in tmpf:
				f.write(l)
		tmpf.close()
		i += 1