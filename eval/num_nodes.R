### num_nodes.R: Computes the number of nodes per crawl, distinguishing between all nodes and only
### only nodes.

####################### CONSTANTS ###################

outDTFile = "plot_data/num_nodes.csv"

####################### HELPER FUNCTIONS ###################

# ## In the first crawls the visitedPeers.csv did not contain IP-Addresses, hence
# ## this wrapper for backwards compatibility
# backwardsCompatible = function(dt) {
#   if (ncol(dt) == 2) {
#     return(nrow(dt[V2 == "true"]))
#   } else {
#     return(nrow(dt[V3 == "true"]))
#   }
# }

## Returns the number of nodes (distinguished by all and only reachable nodes) from a single crawl.
NumNodesSingleCrawl = function(filename) {
  dt = LoadDT(FullPath(filename))
  all = nrow(dt)
  online = nrow(dt[V3 == "true"])
  res = data.table(ts=extractStartDate(filename), all=all, online=online)
  rm(dt)
  return(res)
}

## Calculates the total number of distinct nodes that we've seen during our crawls
NumDistinctNodeTotal = function(crawls) {
  # Idea: go over all crawls, return the IDs and put them in a set
  allDistinctNodes = Reduce(union, pblapply(crawls, function(c) {
    dt = LoadDT(FullPath(c))
    return(dt$V1)
  }))
  return(length(allDistinctNodes))
}

######################### COMPUTATION ##########################

# Source the includes, as always
source("includes.R")

# Get the list of crawl files
crawls = list.files(path=crawlDir, pattern=visitedPattern)
numNodesDT = rbindlist(pblapply(crawls, NumNodesSingleCrawl))

meltedDT = melt(numNodesDT, id=c("ts"))

################## COMPUTATION FOR .TEX VARIABLES ##################

onlineVec = meltedDT[variable == "online"]$value*100/meltedDT[variable == "all"]$value
meanNumNodes = mean(meltedDT[variable == "all"]$value)
numTotalDistinctNodes = NumDistinctNodeTotal(crawls)

writeToEvalRounded("avgOnline", mean(onlineVec), digits=0)
writeToEvalRounded("avgNumberOfNodes", meanNumNodes, digits=0)
writeToEvalRounded("totalDistinctNodes", numTotalDistinctNodes)
# Write the data.table to file to avoid computing it multiple times
write.table(meltedDT, file=outDTFile, sep=";", row.names = F)
