## degree_table.R: Computes the degree of each crawl graph and various other graph metrics.
## Computes:
# 1) Degree distribution table

source("includes.R")

############ CONSTANTS #################

tabOutName = "degree_stats.tex"
tabLabel = "tab:degreeStats"
tabCaption = c("Average of the degree statistics of all \\lraw{numCrawls} crawls.")

crawls = list.files(path=crawlDir, pattern=peerGraphPattern)
########### DEGREE-TABLE ##############

# Load each graph seperately (to avoid crowding the memory) and compute the
# desired metrics about the degree.
# Each call returns a line in the data.table, bind them together with rbindlist

degreeStatsRaw = rbindlist(pblapply(crawls, function(filename) {
  g = loadGraph(FullPath(filename), online=T)

  alldeg = unname(degree(g, v=V(g), mode="total"))
  indeg = unname(degree(g, v=V(g), mode="in"))
  outdeg = unname(degree(g, v=V(g), mode="out"))
  
  dtin = data.table(fname=filename, mode=as.factor("in"), min=min(indeg), mean=mean(indeg), 
                    median=median(indeg), max=max(indeg))
  dtall = data.table(fname=filename, mode=as.factor("total"), min=min(alldeg), mean=mean(alldeg),
                     median=median(alldeg), max=max(alldeg))
  dtout = data.table(fname=filename, mode=as.factor("out"), min=min(outdeg), mean=mean(outdeg),
                     median=median(outdeg), max=max(outdeg))
  dt = rbindlist(list(dtall, dtin, dtout))
  
  return(dt)
}))

# Compute the mean of each column, ignore the filename.
# .SD is the interal subsetted data.table, so the lapply simply
# applies the mean to each column
degreeStats = degreeStatsRaw[, lapply(.SD, mean), .SDcols=-c("fname"), by=c("mode")]

# Round to two digits
degreeStats = degreeStats[, round(.SD, digits=2), by=c("mode")]

### Write out the table (maybe should've used xtable...) ###
# 
# createDegreeTable = function(path, d) {
#   fileConn=path
#   cat(c("\\begin{tabular}{| c | c | c | c | c |}\n", "\\hline\n", " & Min. & Mean & Median & Max.\\\\\n", "\\hline\n"), file=fileConn)
#   
#   for (i in 1:nrow(d)) {
#     cat( c( paste( paste(d[i]$mode, "degree", sep="-"), d[i]$min, d[i]$mean,
#                    d[i]$median, d[i]$max, sep=" & "), "\\\\\n", "\\hline\n"), append=T, file=fileConn)
#   }
#   cat(c("\\end{tabular}\n"), append=T, file=fileConn)
# }

# createDegreeTable(paste(outTabPath, "degree_stats.tex", sep=""), degreeStats)
print(xtable(degreeStats, align = c("|c|c|c|c|c|c|"),
      label=tabLabel,
      caption=tabCaption),
      include.rownames=F,
      hline.after=c(seq(-1, nrow(degreeStats), 1)),
      file=paste(outTabPath, tabOutName, sep=""))
