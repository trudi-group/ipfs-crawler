#!/usr/bin/env python

import os
import sys
import geoip2.database
import ipaddress
import json
import IP2Location
import abc

from maxminddb import (MODE_AUTO, MODE_MMAP, MODE_MMAP_EXT, MODE_FILE,
                       MODE_MEMORY, MODE_FD)

class GeoIPResolver(abc.ABC):

	## expects an IPv4 or IPv6 address as a string, returns a dictionary with keys:
	## IP: LocalIP if argument was a local/private IP, else the provided argument
	## CountryCode: if possible the country code where the IP is from. "NA" for local IPs, "Unkown" in the case of errors.
	## ASNo: if possible the autonomous system number, "NA" for local IPs or in the case of errors
	## ASName: if possible the name of the autonomous system organization, "NA" for local IPs or in the case of errors
	@abc.abstractmethod
	def resolveIP(self, ip):
		pass

class MaxMindResolver(GeoIPResolver):

	def __init__(self, dbPath, countryDBFile = "GeoLite2-Country.mmdb", ASDBFile = "GeoLite2-ASN.mmdb"):
		self.dbPath = dbPath
		self.countryDBFile = countryDBFile
		self.ASDBFile = ASDBFile
		self.countryDB = geoip2.database.Reader(os.path.join(self.dbPath, self.countryDBFile))#, mode=MODE_MMAP_EXT)
		self.ASDB = geoip2.database.Reader(os.path.join(self.dbPath, self.ASDBFile))#, mode=MODE_MMAP_EXT)

	def resolveIP(self, ip):
		# print(ip)
		tmpDict = {}
		if ipaddress.ip_address(ip).is_private or ipaddress.ip_address(ip).is_link_local:
			tmpDict["IP"] = "LocalIP"
			tmpDict["CountryCode"] = "NA"
			tmpDict["ASNo"] = "NA"
			tmpDict["ASName"] = "NA"
		else:
			try:
				countryc = self.countryDB.country(ip).country.iso_code

				if countryc == None:
					countryc = "Unknown"

				resp = self.ASDB.asn(ip)
				tmpDict["IP"] = ip
				tmpDict["CountryCode"] = countryc
				tmpDict["ASNo"] = resp.autonomous_system_number
				tmpDict["ASName"] = resp.autonomous_system_organization

			except geoip2.errors.AddressNotFoundError:
				# Just append "Unknown"
				tmpDict["IP"] = ip
				tmpDict["CountryCode"] = "Unknown"
				tmpDict["ASNo"] = "NA"
				tmpDict["ASName"] = "NA"

		return tmpDict


class IP2LocationResolver(GeoIPResolver):

	def __init__(self, dbPath):
		print("Warning: Autonomous system lookup not implemented yet for IP2Location!")
		self.dbPath = dbPath
		self.IPv6Resolver = IP2Location.IP2Location(os.path.join(self.dbPath, "IP2LOCATION-LITE-DB1.IPV6.BIN"), "SHARED_MEMORY")

	def resolveIP(self, ip):
		tmpDict = {}
		if ipaddress.ip_address(ip).is_private or ipaddress.ip_address(ip).is_link_local:
			tmpDict["IP"] = "LocalIP"
			tmpDict["CountryCode"] = "NA"
			tmpDict["ASNo"] = "NA"
			tmpDict["ASName"] = "NA"
		else:
			resolv = self.IPv6Resolver

			resp = resolv.get_all(ip)
			tmpDict["IP"] = ip
			tmpDict["CountryCode"] = resp.country_short
			tmpDict["ASNo"] = "NA"
			tmpDict["ASName"] = "NA"

		return tmpDict


#### Helper Functions ####

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
	
	# onlyOnlineNodes = Falseprint
	geoIPPath = "geoipDBs/"
	resolver = MaxMindResolver(geoIPPath)
	# resolver = IP2LocationResolver(geoIPPath)

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

		crawldata = {}

		with open(crawlDir+crawlFile, "r") as f:
			crawldata = json.load(f)

			for node in crawldata["Nodes"]:
				# Roadmap: 
				# * Take each entry and lookup the country code, AS-no and AS name for each IP
				# * Put everything back together
				# * Write back to *the same* file (through a temp-file)

				multiaddrs = node["MultiAddrs"]
				# We had at least one occurence of an ID without address -> ignore that
				if len(multiaddrs) == 0:
					print("No addresses to peer ID " + node["NodeID"])
					continue

				## TODO: Properly split the addresses. libp2p-py seems a bit outdated and unstable, but maybe it can be of value?
				MAProtArray = [ParseMA(ma) for ma in multiaddrs]
				# We get (protocol, addr) back, if the address is ipv4 or ipv6
				AddrArray = list(set([ma[1] for ma in MAProtArray if ma[1]]))

				# We now proceed with the lookup and enhance the simple list of multiaddrs by more detailed info:
				tmpAddrLst = []
				for ip in AddrArray:
					tmpDict = resolver.resolveIP(ip)
					tmpAddrLst.append(tmpDict)

				node["MultiAddrs"] = tmpAddrLst
				# print(crawldata["Nodes"][0])

		with open(outputDir+crawlFile, "w", encoding="utf-8") as f:
			json.dump(crawldata, f)
		i += 1