## plot_sessionLengths.R

source("includes.R")

dataPath = "plot_data/session_lengths.csv"
outTablePath = "tables/session_lengths.tex"
plotUnits = "mins"
# plotUnits = "days"
# plotUnits = "hours"
cutOff = 0.99
## We load the average crawl duration from the variables_for_tex.lua file.
## This dependency is reflected in the Makefile
minimumCutOff = as.difftime(GetFromEval("avgCrawlDuration"), units="mins")
units(minimumCutOff) = plotUnits
minimumCutOff = as.numeric(minimumCutOff)

## Load the data, timedurations are in seconds
dt = LoadDT(dataPath)
setnames(dt, 2, "timediff")
duration = as.difftime(dt$timediff, unit="secs")

totalNumberOfSessions = length(duration)
writeToEvalRounded("totalNumberOfSessions", totalNumberOfSessions)

## Convert the duration to what we'll use throughout the analysis
units(duration) = plotUnits

## We will use the ecdf later for a summary table
empDistr = ecdf(duration)

## Also, compute the cutoff value, everyhing above this percentile
# will be ignored for the sake of readability in the plot.

cutOffVal = as.numeric(quantile(duration, cutOff))

## Note how many sessions had length 0, i.e., only occured once
#duration[duration == 0]

## Summarize the counts in a data.table
duration = as.numeric(duration)


### OUTPUT TABLE

## We want to have the number of sessions **longer** than a certain period as well as their
## percentage in terms of all measured sessions.
## R's difftime is nice enough to make this very convenient for us

cumulativeValues = c(
  as.difftime(5, unit="mins"),
  as.difftime(10, unit="mins"),
  as.difftime(30, unit="mins"),
  as.difftime(1, unit="hours"),
  as.difftime(1, unit="days"),
  as.difftime(6, unit="days")
)
## Fixme: Not the nicest way to duplicate this here, but oh well.
tableVals = c(
  "5 minutes",
  "10 minutes",
  "30 minutes",
  "1 hour",
  "1 day",
  "6 days"
)
units(cumulativeValues) = plotUnits

numSessionsAbove = sapply(cumulativeValues, function(x) {
  length(duration[duration >= x])
})

percentageAbove = round(100*numSessionsAbove/totalNumberOfSessions, digits=4)

## Gather them all in a data.table

tableDT = data.table(value=tableVals, percent=percentageAbove, numSessions=numSessionsAbove)

createSessionLengthTable = function(path, d) {
  fileConn=path
  cat(c("\\begin{tabular}{| c | c | c |}\n", "\\hline\n", "Session Duration & Percentage & Number of sessions\\\\\n", "\\hline\n"), file=fileConn)
  
  for (i in 1:nrow(d)) {
    cat( c( paste( d[i]$value, d[i]$percent, d[i]$numSessions, sep=" & "), "\\\\\n", "\\hline\n"), append=T, file=fileConn)
  }
  cat(c("\\end{tabular}\n"), append=T, file=fileConn)
}

createSessionLengthTable(outTablePath, tableDT)
