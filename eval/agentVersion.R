source("includes.R")

##### CONSTANTS #####

outDataFile = "plot_data/agent_versions.csv"

# Get the list of crawl files
crawls = list.files(path=crawlDir, pattern=visitedPattern)

## Load the data, set the appropriate name for the version column and merge the data.table by version
allDTs = rbindlist(pblapply(crawls, function(filename) {
  tmpDT = LoadDT(FullPath(filename), header=F)
  setnames(tmpDT, 4, "version")
  return(tmpDT[version != "", .(count= .N), .(version)])
}))

agentCounts = allDTs[, .(avgcount = mean(count)), by="version"]
agentCounts = agentCounts[order(-avgcount)]

write.table(agentCounts, file=outDataFile, sep=";", row.names=F)