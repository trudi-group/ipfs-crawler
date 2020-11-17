### num_nodes.R: Computes the number of nodes per crawl, distinguishing between all nodes and only
### only nodes.

####################### CONSTANTS ###################

outDTFile = "plot_data/num_nodes.csv"

####################### HELPER FUNCTIONS ###################

## Returns the number of nodes (distinguished by all and only reachable nodes) from a single crawl.
NumNodesSingleCrawl = function(filename) {
  dt = LoadDT(FullPath(filename))
  all = nrow(dt)
  setnames(dt, 3, "online")
  online = nrow(dt[online == T])
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

# Crawls should not be more than this [in seconds] apart to be classified 
# as stemming from one measurement
interCrawlDistance = 3600

# Get the list of crawl files
crawls = list.files(path=crawlDir, pattern=visitedPattern)
numNodesDT = rbindlist(pblapply(crawls, NumNodesSingleCrawl))
numNodesDT = numNodesDT[order(ts)]

samecrawls = c(FALSE, diff(as.numeric(numNodesDT$ts)) > interCrawlDistance)
numNodesDT$crawlno = cumsum(samecrawls)

dates = numNodesDT[seq(1, nrow(numNodesDT), 3)]$ts

meltedDT = setDT(melt(numNodesDT, id=c("crawlno", "ts")))

write.table(meltedDT, file="plot_data/raw_num_nodes.csv", sep=";", row.names = F, append = T, col.names = F)

meltedDT$ts = NULL
meltedDT = meltedDT[, .(avgcount = mean(value)), by=.(variable, crawlno)]
# meltedDT$ts = rep(extractStartDate(crawls[1]), nrow(meltedDT))
## Not beautiful but functional
meltedDT$ts = as.POSIXct(c(dates, dates), origin="1970-01-01")
meltedDT$crawlno = NULL
setcolorder(meltedDT, c("ts", "variable", "avgcount"))

# Write the data.table to file to avoid computing it multiple times
write.table(meltedDT, file=outDTFile, sep=";", row.names = F, append = T, col.names = F)
