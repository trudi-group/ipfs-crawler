### protocolStats.R: outputs basic statistics about the used protocols in the found multiaddresses.
### The data is assumed to reside in a .csv where it is simply read and output into a table.
source("includes.R")

######### CONSTANTS ###########

sourceFile = "plot_data/protocol_stats.csv"

########## FUNCTIONS ###########

createProtocolTable = function(path, inputdt) {
  setnames(inputdt, c("prot", "count", "perc"))
  fileConn=path
  cat(c("\\begin{tabular}{| c | c | c |}\n", "\\hline\n", "Protocol & Perc. of peers & Abs. count\\\\\n", "\\hline\n"),
      file=fileConn)
  for(i in seq(1, nrow(inputdt), by=1)) {
    cat(c(paste(inputdt[i]$prot, round(inputdt[i]$perc, digits=4), inputdt[i]$count, sep=" & "), "\\\\\n", "\\hline\n"),
        append=T, file=fileConn)
  }
  cat(c("\\end{tabular}\n"), append=T, file=fileConn)
}

####### COMPUTATION ##############

dt = LoadDT(sourceFile, header=T)
countDT = dt[, .(count = .N), .(protocol)]
numPeers = length(unique(dt$nodeid))
countDT$percentage = countDT$count*100/numPeers
# dt$count = NULL

createProtocolTable(paste(outTabPath, "protocol_stats.tex",sep=""), countDT)