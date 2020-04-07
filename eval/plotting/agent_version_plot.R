source("includes.R")

##### CONSTANTS #####

clientCutOff = 10
inDataFile = "plot_data/agent_versions.csv"
tabOutName = "full_agent_version_tab.tex"
tabLabel = "tab:full_agent_version_tab"
tabCaption = "The full list of agent versions and how often their were seen on average per crawl."

##### PROCESSING & PLOTTING #####

## Load the data
agentCounts = LoadDT(inDataFile, header=T)

## To ease presentation, we only focus on the top clientCutOff versions
truncatedDT = agentCounts[1:clientCutOff]

## The top clientCutOff-versions make for this many of all seen clients:
totalNumberOfVersions = sum(agentCounts$avgcount)
truncatedVersions = sum(truncatedDT$avgcount)
writeToEvalRounded("AgentVersionIncludeTruncatedPercentage", truncatedVersions*100/totalNumberOfVersions)
# print(sum(truncatedDT$avgcount)/totalNumberOfVersions)

## Reorder the versions so that the plot labels are in decreasing order
truncatedDT = truncatedDT[, pos:=cumsum(avgcount)-0.5*avgcount, by="version"]
truncatedDT$version = with(truncatedDT, reorder(version, -avgcount))

q = ggplot(truncatedDT, aes(x="", y=avgcount, fill=version)) +
  geom_bar(width=1, stat="identity", color="white", position="dodge") +
  xlab("") + ylab("Average count per crawl") +
  scale_y_continuous(breaks = scales::pretty_breaks(n = plotBreakNumber))

## Output to .tex but also to .png
# tikz(file=paste(outPlotPath, "agent_version_distribution.tex", sep=""), width=plotWidth, height=plotHeight)
# q
# dev.off()

# pdf(file=paste(outPlotPath, "agent_version_distribution.pdf", sep=""), width=bitmapWidth, height=bitmapHeight)
# q
# dev.off()

png(filename=paste(outPlotPath, "agent_version_distribution.png", sep=""), height = bitmapHeight, width=bitmapWidth)
q
dev.off()

## Output a complete table of all agent versions

print(xtable(agentCounts, align = c("|c|l|c|"),
             label=tabLabel,
             caption=tabCaption),
      tabular.environment="longtable",
      floating=F,
      include.rownames=F,
      hline.after=c(seq(-1, nrow(agentCounts), 1)),
      file=paste(outTabPath, tabOutName, sep="")
)