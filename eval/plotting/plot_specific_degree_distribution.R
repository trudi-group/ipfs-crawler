## plot_specific_degree_distribution.R: Plots the degree distribution of a specific crawl
## The command line argument (if specified) yields the index of the crawl.
## Default is 1, i.e., the first crawl of the set.

args = commandArgs(trailingOnly = T)
if(length(args) == 1) {
  graphChoiceIndex = as.numeric(args[1])
} else {
  graphChoiceIndex = 1
}

print(paste("Using graph with index ", graphChoiceIndex, sep=""))

########################## HELPER FUNCTION ####################

computeDegreeDistribution = function(graph, mode="in") {
  # This igraph-function returns a vector from 0 to max(degree), so a lot of
  # the higher degree entries are actually not found in our graph but distort
  # the plotting (and make the log-axis look bad). So we will eliminate the 0 entries
  # before plotting.
  degDistr = degree_distribution(graph, v = V(graph), mode=mode)
  # Only use the degrees we actually found
  existentDegrees = degDistr != 0
  # Since R starts with index 1, the igraph function has degree 0 at index 1
  # -> Have to do an index shift
  allDegrees = seq(0, length(degDistr) - 1, by=1)
  # This works because "True" = 1
  dt = data.table(degree=allDegrees[which(existentDegrees)], freq=degDistr[which(existentDegrees)], 
                  mode=rep(mode, sum(existentDegrees)))
  return(dt)
}

########################## PLOTTING ####################

# Get the includes (as always)
source("includes.R")

# Plot only in-degree?
onlyIn = T

# Get a list of all graph files
crawls = list.files(path=crawlDir, pattern=peerGraphPattern)

## CONSTANTS ##
graphChoice = crawls[[graphChoiceIndex]]

allgraph = loadGraph(FullPath(graphChoice), online=F)
onlinegraph = loadGraph(FullPath(graphChoice), online=T)

tmpDTAllGraphIn = computeDegreeDistribution(allgraph, mode="in")
tmpDTAllGraphOut = computeDegreeDistribution(allgraph, mode="out")
tmpDTAllGraph = rbindlist(list(tmpDTAllGraphIn, tmpDTAllGraphOut))

tmpDTOnlineGraphIn = computeDegreeDistribution(onlinegraph, mode="in")
tmpDTOnlineGraphOut = computeDegreeDistribution(onlinegraph, mode="out")
tmpDTOnlineGraph = rbindlist(list(tmpDTOnlineGraphIn, tmpDTOnlineGraphOut))


## Enhance the data.tables with a type in case we want to combine them
# tmpDTAllGraph$type = rep("all", nrow(tmpDTAllGraph))
# tmpDTOnlineGraph$type = rep("online", nrow(tmpDTOnlineGraph))
# dt = rbindlist(list(tmpDTAllGraph, tmpDTOnlineGraph))


### PLOTTING ###

if (onlyIn) {
  tmpDTOnlineGraph = tmpDTOnlineGraph[mode=="in"]
  tmpDTAllGraph = tmpDTAllGraph[mode=="in"]
  
  qAll = ggplot(tmpDTAllGraph, aes(x=degree, y=freq)) + geom_point() +
    xlab("Degree") + ylab("Relative frequency") +
    scale_y_log10() + scale_x_log10()
  
  qOnline = ggplot(tmpDTOnlineGraph, aes(x=degree, y=freq)) + geom_point() +
    xlab("Degree") + ylab("Relative frequency") +
    scale_y_log10() + scale_x_log10()
  
} else {
  
  qAll = ggplot(tmpDTAllGraph, aes(x=degree, y=freq, color=mode, shape=mode)) + geom_point() +
    xlab("Degree") + ylab("Relative frequency") +
    scale_color_discrete(name="Degre type", labels=c("In", "Out")) +
    scale_shape_discrete(name="Degre type", labels=c("In", "Out")) +
    scale_y_log10() + scale_x_log10()
  
  qOnline = ggplot(tmpDTOnlineGraph, aes(x=degree, y=freq, color=mode, shape=mode)) + geom_point() +
    xlab("Degree") + ylab("Relative frequency") +
    scale_color_discrete(name="Degre type", labels=c("In", "Out")) +
    scale_shape_discrete(name="Degre type", labels=c("In", "Out")) +
    scale_y_log10() + scale_x_log10()
}


# tikz(file=paste(outPlotPath, "log_all_nodes_degree_distribution.tex", sep=""), width=plotWidth, height=plotHeight)
# qAll
# dev.off()
# 
# tikz(file=paste(outPlotPath, "log_online_nodes_degree_distribution.tex", sep=""), width=plotWidth, height=plotHeight)
# qOnline
# dev.off()

png(filename=paste(outPlotPath, "log_all_nodes_degree_distribution.png", sep=""), height = bitmapHeight, width=bitmapWidth)
qAll
dev.off()

png(filename=paste(outPlotPath, "log_online_nodes_degree_distribution.png", sep=""), height = bitmapHeight, width=bitmapWidth)
qOnline
dev.off()

# pdf(file=paste(outPlotPath, "log_all_nodes_degree_distribution.pdf", sep=""), width=bitmapWidth, height=bitmapHeight)
# qAll
# dev.off()
# 
# pdf(file=paste(outPlotPath, "log_online_nodes_degree_distribution.pdf", sep=""), width=bitmapWidth, height=bitmapHeight)
# qOnline
# dev.off()



######################## BY-HAND COMPUTATION OF DEGREE DISTRIBUTION ########

# plotDegreeDistribution = function(graph, mode="in", log=F) {
#   # Roadmap
#   
#   degs = degree(graph, V(graph), mode=mode)
#   tmpdt = data.table(table(degs))
#   tmpdt$degs = as.numeric(tmpdt$degs)
#   # Convert to relative frequencies
#   tmpdt$N = tmpdt$N / gorder(graph)
#   
#   degsout = degree(graph, V(graph), mode="out")
#   tmpoutDT = data.table(table(degsout))
#   tmpoutDT$degsout = as.numeric(tmpoutDT$degsout)
#   tmpoutDT$N = tmpdt$N / gorder(graph)
#   
#   mergedDT = merge(tmpdt, tmpoutDT, by.x=c("degs"), by.y=c("degsout"), all=T)
#   meltedDT = melt(mergedDT, id=c("degs"))
#   q = ggplot(meltedDT, aes(x=degs, y=value, color=variable, shape=variable)) + 
#     geom_point(size=plotPointSize, na.rm=T) +
#     xlab("Degree") + ylab("Relative Frequency") +
#     scale_shape_discrete(name="Degree", labels=c("In", "Out")) + 
#     scale_color_discrete(name="Degree", labels=c("In", "Out"))
#   if(log) {
#     q = q + scale_y_log10() + scale_x_log10()
#   }
#   return(q)
# }