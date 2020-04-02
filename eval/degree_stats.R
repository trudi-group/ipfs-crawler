## degree_table.R: Computes the degree of each crawl graph and various other graph metrics.
## Computes:
# 1) Degree distribution table

source("includes.R")

############ CONSTANTS #################

outDataFile = "plot_data/degree_stats.csv"

########### DEGREE-TABLE DATA ##############

# Load each graph seperately (to avoid crowding the memory) and compute the
# desired metrics about the degree.
# Each call returns a line in the data.table, bind them together with rbindlist

crawls = list.files(path=crawlDir, pattern=peerGraphPattern)

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

write.table(degreeStats, file=outDataFile, sep=";", row.names=F)