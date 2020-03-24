source("includes.R")

##### CONSTANTS #####
clientCutOff = 10

# Get the list of crawl files
crawls = list.files(path=crawlDir, pattern=visitedPattern)

## Load the data, set the appropriate name for the version column and merge the data.table by version
allDTs = rbindlist(pblapply(crawls, function(filename) {
  tmpDT = LoadDT(FullPath(filename), header=F)
  setnames(tmpDT, 4, "version")
  return(tmpDT[version != "", .(count= .N), .(version)])
}))

agentCounts = allDTs[, .(avgcount = mean(count)), by="version"]

## To ease presentation, we only focus on the top clientCutOff versions
truncatedDT = agentCounts[order(-avgcount)][1:clientCutOff]

## The top clientCutOff-versions make for this many of all seen clients:
totalNumberOfVersions = sum(agentCounts$avgcount)
writeToEvalRounded("AgentVersionIncludeTruncatedPercentage", sum(truncatedDT$avgcount)*100/totalNumberOfVersions)
print(sum(truncatedDT$avgcount)/totalNumberOfVersions)

## Reorder the versions so that the plot labels are in decreasing order
truncatedDT = truncatedDT[, pos:=cumsum(avgcount)-0.5*avgcount, by="version"]
truncatedDT$version = with(truncatedDT, reorder(version, -avgcount))


q = ggplot(truncatedDT, aes(x="", y=avgcount, fill=version)) +
  geom_bar(width=1, stat="identity", color="white") +
  coord_polar("y", start=0) +
  geom_text(aes(label = avgcount), position = position_stack(vjust = 0.5), color = "white")+
  theme_void()

## Output to .tex but also to .png
tikz(file=paste(outPlotPath, "agent_version_distribution.tex", sep=""), width=plotWidth, height=plotHeight)
q
dev.off()

png(filename=paste(outPlotPath, "agent_version_distribution.png", sep=""), height = bitmapHeight, width=bitmapWidth)
q
dev.off()

