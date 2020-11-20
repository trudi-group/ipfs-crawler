source("includes.R")

##### CONSTANTS #####

clientCutOff = 10
inDataFile = "plot_data/agent_versions.csv"
# tabOutName = "full_agent_version_tab.tex"
# tabLabel = "tab:full_agent_version_tab"
# tabCaption = "The full list of agent versions and how often their were seen on average per crawl."

##### PROCESSING & PLOTTING #####

## Load the data
agentCounts = LoadDT(inDataFile, header=F)
setnames(agentCounts, 1:3, c("ts", "version", "avgcount"))

## The top clientCutOff-versions make for this many of all seen clients:
# totalNumberOfVersions = sum(agentCounts$avgcount)
# truncatedVersions = sum(truncatedDT$avgcount)
# writeToEvalRounded("AgentVersionIncludeTruncatedPercentage", truncatedVersions*100/totalNumberOfVersions)
# print(sum(truncatedDT$avgcount)/totalNumberOfVersions)

## Reorder the versions so that the plot labels are in decreasing order
truncatedDT = agentCounts
truncatedDT$version = with(truncatedDT, reorder(version, -avgcount))

truncatedDT$ts = as.POSIXct(truncatedDT$ts)

nonTruncated = truncatedDT
lastCrawl = nonTruncated[ts == max(ts, na.rm=T)]
lastCrawl$ts = NULL
pieCutoff = 10
aggrLastCrawl = rbindlist(list(lastCrawl[1:pieCutoff], data.table(version="Others", avgcount=sum(lastCrawl[-(1:pieCutoff)]$avgcount))))

wideav = transpose(aggrLastCrawl, make.names="version")
write.table(wideav, "plot_data/last_crawl_av.csv", row.names=F, sep=",")

## To ease presentation, we only focus on the top clientCutOff versions
truncatedDT = truncatedDT[, .SD[1:clientCutOff], by="ts"]


q = ggplot(truncatedDT, aes(x=ts, y=avgcount, color=version, linetype=version)) + 
  geom_line() + geom_point() +
  xlab("Timestamp") + ylab("Average count") +
  scale_y_continuous(breaks = scales::pretty_breaks(n = plotBreakNumber)) +
  scale_x_datetime(date_labels = plotDateFormat) +
  scale_linetype_discrete(name="Version") +
  scale_color_discrete(name="Version") +
  theme(axis.text.x = element_text(angle = 45, vjust = 1, hjust=1))

png(filename=paste(outPlotPath, "agent_version_distribution.png", sep=""), height = bitmapHeight, width=bitmapWidth)
q
dev.off()

write.table(truncatedDT, "plot_data/agent_versions_truncated.csv", row.names=F, sep=";")
## Output a complete table of all agent versions

# print(xtable(agentCounts, align = c("|c|l|c|"),
#              label=tabLabel,
#              caption=tabCaption),
#       tabular.environment="longtable",
#       floating=F,
#       include.rownames=F,
#       hline.after=c(seq(-1, nrow(agentCounts), 1)),
#       file=paste(outTabPath, tabOutName, sep="")
# )
