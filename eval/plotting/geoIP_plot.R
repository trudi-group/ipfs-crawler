## geoIP_plot.R. Plots the geoIP data from geoIP.R

source("includes.R")

inputDataFile = "plot_data/geoIP_per_crawl.csv"
tabPath = "./tables/geoIP_statistics.tex"
completeTabPath = "./tables/all_countries_geoIP_statistics.tex"

countryCutOff = 10
## Numbers <= 12 are written out
if (countryCutOff <= 12) {
  writeToEval("topNNodes", wordNumbers[countryCutOff])
} else {
  writeToEvalRounded("topNNodes", countryCutOff)
}

mcounts = LoadDT(inputDataFile, header=T)
mcounts$N = as.double(mcounts$N)
setnames(mcounts, 1:4, c("CountryCode", "N", "timestamp", "type"))

numNodes = mcounts[type=="all", .(numNodes = sum(N)), .(timestamp)]
numNodesReachable = mcounts[type=="reachable", .(numNodes = sum(N)), .(timestamp)]
localIPs = mcounts[CountryCode == "LocalIP",][,N,timestamp]
avgPercLocalIP = mean(100*localIPs$N/numNodes$numNodes)
writeToEvalRounded("PercLocalIPs", avgPercLocalIP)

numberOfTimestamps = length(unique(mcounts$timestamp))
## The mean + CI computation
aggrCounts = mcounts[, .(AvgCount = sum(N)/numberOfTimestamps, 
                         CILower = format(max(CI(N)[1], 0), scientific = F), 
                         CIUpper = CI(N)[2]), by = .(CountryCode, type)]

## Sort by avgcount, so we can take the 10 (or countryCutoff) largest shares
mcountsAll = aggrCounts[order(-AvgCount)]
mcountsReachable = aggrCounts[type=="reachable"]
mcountsReachable = mcountsReachable[order(-AvgCount)]

## For eval.lua: How big is the share of the top N countries?
topNAllCountries = mcountsAll[type == "all"][1:10]$CountryCode
DTTopNAll = mcounts[mcounts[type=="all", .I[CountryCode %in% topNAllCountries], .(timestamp)][,V1]]
DTTopNAll = DTTopNAll[, .(sumTopN = sum(N)), .(timestamp)]
merged = DTTopNAll[numNodes, on=c("timestamp")]
topNAllPercentage = merged$sumTopN*100/merged$numNodes

writeToEvalRounded("geoIPtopNAllPercentage", mean(topNAllPercentage))

## The same for the reachable nodes
topNReachableCountries = mcountsReachable[1:10]$CountryCode

DTTopNReachable = mcounts[mcounts[type=="reachable", .I[CountryCode %in% topNReachableCountries], .(timestamp)][,V1]]
DTTopNReachable = DTTopNReachable[, .(sumTopN = sum(N)), .(timestamp)]
mergedReachable = DTTopNReachable[numNodesReachable, on=c("timestamp")]
topNReachablePercentage = mergedReachable$sumTopN*100/mergedReachable$numNodes
# reachableStats = data.table(avg=mean(topNReachablePercentage),
                            # CILower=CI(topNReachablePercentage)[1], CIUpper=CI(topNReachablePercentage[2]))
writeToEvalRounded("geoIPtopNReachablePercentage", mean(topNReachablePercentage))

DTTopNAll$sumTopN/numNodes$numNodes

##### PLOTTING #########
# plotDT = rbindlist(list(mcountsAll[1:countryCutOff], mcountsReachable[1:countryCutOff]))
allPlotDT = mcountsAll[type == "all"][1:countryCutOff]
ReachablePlotDT = mcountsReachable[1:countryCutOff]
## The number of occurence is the average over all crawls


# Screw xtable, it's too much of a hassle to get to work properly
createTopNTable = function(path, topNdataAll, topNDataReachable, longtable=F) {
  setnames(topNdataAll, c("country", "type", "AvgCount", "CILower", "CIUpper"))
  setnames(topNDataReachable, c("country", "type", "AvgCount", "CILower", "CIUpper"))
  fileConn=path
  if (longtable) {
    cat(c("\\begin{longtable}{| c | c | c || c | c | c |}\n",
          "\\hline\n",
          "\\multicolumn{3}{| c ||}{All} & \\multicolumn{3}{| c |}{Reachable}\\\\\n",
          "\\hline\n", "Country & Count & Conf. Int. & Country & Count & Conf. Int.\\\\\n", "\\hline\n"), file=fileConn)
  } else {
    cat(c("\\begin{tabular}{| c | c | c || c | c | c |}\n",
          "\\hline\n",
          "\\multicolumn{3}{| c ||}{All} & \\multicolumn{3}{| c |}{Reachable}\\\\\n",
          "\\hline\n", "Country & Count & Conf. Int. & Country & Count & Conf. Int.\\\\\n", "\\hline\n"), file=fileConn)
  }
  
  for(i in seq(1, nrow(topNdataAll), by=1)) {
    cat(c(paste(
                topNdataAll[i]$country,
                round(topNdataAll[i]$AvgCount, digits=2),
                paste("$\\pm$", round(topNdataAll[i]$CIUpper - topNdataAll[i]$AvgCount, digits=2), sep=""),
                topNDataReachable[i]$country,
                round(topNDataReachable[i]$AvgCount, digits=2),
                paste("$\\pm$", round(topNDataReachable[i]$CIUpper - topNDataReachable[i]$AvgCount, digits=2), sep=""),
                sep=" & "), "\\\\\n", "\\hline\n"), append=T, file=fileConn)
    # writeLines(c(paste(topNdata[i]$country, topNdata[i]$value, sep=" & "), "\\\\"), fileConn)
  }
  # cat(c("\\hhline{|=|=|}\n", paste("Sum", round(sum(topNdata$value), digits=2), sep=" & "), "\\\\\n"), append=T, file=fileConn)
  if (longtable) {
    cat(c("\\end{longtable}\n"), append=T, file=fileConn)
  } else {
    cat(c("\\end{tabular}\n"), append=T, file=fileConn)
  }
}

createTopNTable(tabPath, allPlotDT, ReachablePlotDT)

## And the complete table for the appendix
createTopNTable(completeTabPath, mcountsAll[type == "all"], mcountsReachable, longtable=T)
