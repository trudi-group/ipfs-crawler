source("includes.R")

##### CONSTANTS #####

outDataFile = "plot_data/agent_versions.csv"

# Get the list of crawl files
crawls = list.files(path=crawlDir, pattern=visitedPattern)

## Load the data, set the appropriate name for the version column and merge the data.table by version
allDTs = rbindlist(pblapply(crawls, function(filename) {
  tmpDT = LoadDT(FullPath(filename), header=F)
  setnames(tmpDT, 4, "version")
  aggrDT = tmpDT[version != "", .(count= .N), .(version)]
  aggrDT$ts = rep(extractStartDate(filename), nrow(aggrDT))
  rm(tmpDT)
  return(aggrDT)
}))

allDTs = allDTs[order(ts)]

## In seconds
interCrawlDistance = 3600
samecrawls = c(FALSE, diff(as.numeric(allDTs$ts)) > interCrawlDistance)
crawlno = cumsum(samecrawls)

allDTs$crawlno = crawlno

write.table(allDTs, "agent_version_raw.csv", col.names = T, append =T, row.names=F, sep = ";")
## There are thre crawls per cron job -> associate crawlnumbers with timestamps, so we can ditch the old ts and crawlno
allDTs$nts = sort(unique(allDTs$ts))[(allDTs$crawlno + 1)*3]

allDTs$ts = NULL
allDTs$crawlno = NULL

agentCounts = allDTs[, .(avgcount = mean(count)), .(version, nts)]
agentCounts = agentCounts[order(nts, -avgcount)]
setcolorder(agentCounts, c("nts", "version", "avgcount"))
setnames(agentCounts, 1, "date")

write.table(agentCounts, file=outDataFile, sep=";", row.names=F, append = T, col.names = F)
