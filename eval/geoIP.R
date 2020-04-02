source("includes.R")

########### CONSTANTS #################

outDataFile = "plot_data/geoIP_per_crawl.csv"

processedPeerFiles = "../output_data_crawls/geoIP_processing/"

procCrawls = list.files(processedPeerFiles, pattern=visitedPattern)

CountsPerTS = pblapply(procCrawls, function(pc) {
  fdate = extractStartDate(pc)
  dat = LoadDT(paste(processedPeerFiles, pc, sep=""), header=T)
  dat$ASNO = NULL
  dat$ASName = NULL
  dat$IP = NULL
  dat$agentVersion = NULL
  ## All peers with only LocalIP
  # localIPIndexSet = dat[, .I[.N == 1 && grepl("LocalIP", country, fixed=T)], .(nodeid)][,V1]
  localIPIndexSet = dat[, .I[all(grepl("LocalIP", country, fixed=T))], .(nodeid)][,V1]
  numLocalIPs = length(unique(dat[localIPIndexSet]$nodeid))
  ## We want:
  ## * Count the country if there is one and ignore the LocalIP
  ## * Take the country with the majority (solve ties)
  ## * We excluded the localIPs already
  ## So let's first count the countries for each nodeid
  countryCount = dat[(!grepl("LocalIP", country, fixed=T)), .(count =.N), by=c("nodeid", "country", "online")]
  ## Enter some data.table magic: For each ID, we want the row that
  ## has the maximum count. .I gives the index in the original data.table
  ## that fulfills the expression for a given ID.
  ## This yields a vector of countries which we count with table() and
  ## give the result back to data.table
  ccTmp = countryCount[countryCount[, .I[count == max(count)], by=c("nodeid")][,V1]]
  ## We resolve duplicates by just taking the first value
  ccTmp = ccTmp[ccTmp[, .I[1], .(nodeid, country, online)][,V1]]
  
  tabAll = data.table(table(ccTmp$country))
  tabAll = rbindlist(list(tabAll, data.table(V1 = c("LocalIP"), N = c(numLocalIPs))))
  tabAll$timestamp = rep(fdate, nrow(tabAll))
  tabAll$type = rep("all", nrow(tabAll))
  tabReachable = data.table(table(ccTmp[online=="true"]$country))
  tabReachable$timestamp = rep(fdate, nrow(tabReachable))
  tabReachable$type = rep("reachable", nrow(tabReachable))
  # tmpDT = data.table(table(countryCount[countryCount[, .I[count == max(count)], by=c("nodeid")][,V1]]$country))
  # tmpDT$timestamp = rep(fdate, nrow(tmpDT))
  return(rbindlist(list(tabAll, tabReachable)))
})

## Combine the data tables into one and take the mean+conf int. We deliberately use the number of
## "observations" in terms of time stamps, to not distort the picture.
## By looking at this from a per-crawl-perspective, we avoid over-representation of
## always-on peers. This could happen if we looked at absolute numbers as before.

mcounts = rbindlist(CountsPerTS)
mcounts$N = as.double(mcounts$N)

write.table(mcounts, file=outDataFile, sep=";", row.names=F)
