source("includes.R")

##### CONSTANTS #####

inDataName = "plot_data/degree_stats.csv"
tabOutName = "degree_stats.tex"
tabLabel = "tab:degreeStats"
tabCaption = c("Average of the degree statistics of all \\lraw{numCrawls} crawls.")

##### PROCESSING & OUTPUT #####

degreeStats = LoadDT(inDataName, header=T)

print(xtable(degreeStats, align = c("|c|c|c|c|c|c|"),
             label=tabLabel,
             caption=tabCaption),
      include.rownames=F,
      hline.after=c(seq(-1, nrow(degreeStats), 1)),
      file=paste(outTabPath, tabOutName, sep=""))