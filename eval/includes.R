# Includes, libraries and constants

packages = c("data.table", "reshape2", "ggplot2", "scales",
             "tikzDevice", "stringr", "pbapply", "igraph", "xtable", "jsonlite", "tidyr")

# shamelessly copied from Github (https://gist.github.com/stevenworthington/3178163)
# It loads all required packages and installs them if they're not present
ipak <- function(pkg){
  new.pkg <- pkg[!(pkg %in% installed.packages()[, "Package"])]
  if (length(new.pkg)) 
    install.packages(new.pkg, dependencies = TRUE)
  sapply(pkg, function(p) {
    suppressPackageStartupMessages(require(p, character.only = TRUE))
  })
}

ipak(packages)
pbo = pboptions(type="txt") # !!!
theme_set(theme_bw(10))

################# CONSTANTS ###########################

crawlDir = "../output_data_crawls/"
processedPeerFiles = "../output_data_crawls/geoIP_processing/"
outFilePath = "variables_for_tex.lua"
outPlotPath = "./figures/"
outTabPath = "./tables/"
visitedPattern = "visitedPeers_[:alphanum:]*"
peerGraphPattern = "peerGraph_[:alphanum:]*"
## In the current version of the JSON file, the node array comes at position 3
jsonNodeContentIndex = 3
plotWidth = 3.5
plotHeight = 2.5
bitmapHeight = 860
bitmapWidth = 1200
plotDateFormat = "%d/%m %H:%M"
plotPointSize = 0.5
palette <- c("#018571", "#DFC27D", "#A6611A")
plotBreakNumber = 5

wordNumbers = c("\"one\"", "\"two\"", "\"three\"", "\"four\"", "\"five\"", "\"six\"", "\"seven\"",
                "\"eight\"", "\"nine\"", "\"ten\"", "\"eleven\"", "\"twelve\"")
################## HELPER FUNCTIONS ################

## Just a quick wrapper around paste(...)
FullPath = function(filename) {
  return(paste(crawlDir, filename, sep=""))
}

## Writes an entry in the eval file of the form <name>=<var>
## cat opens the file for the duration of the function call
## It will first erase all occurences of the variable before writing the new entry.
writeToEval = function(name, var) {
  if(!file.exists(outFilePath) && !dir.exists(outFilePath)) {
    file.create(outFilePath)
  }
  RMFromEval(name)
  cat(paste(paste(name, var, sep="="), "\n", sep=""), file=outFilePath, append=T)
}

## Write to eval file but round to n digits (default: 2)
writeToEvalRounded = function(name, var, digits=2) {
  writeToEval(name, round(var, digits=digits))
}

## Extracts the *start* date of the crawl
extractStartDate = function(filename, tz="CET") {
  cdatestr = strsplit(filename, "_")[[1]][2]
  cdate = as.POSIXct(strptime(cdatestr, format="%d-%m-%y--%H:%M:%S"))
  return(cdate)
}

## Computes the duration of the crawl from the timestamps in the filename
crawlDurationFromFilename = function(filename) {
  split = strsplit(filename, "_")
  startStr = split[[1]][2]
  startDate = as.POSIXct(strptime(startStr, format="%d-%m-%y--%H:%M:%S"))
  endStr = strsplit(split[[1]][3], "\\.")[[1]][1]
  endDate = as.POSIXct(strptime(endStr, format="%d-%m-%y--%H:%M:%S"))
  return(as.double(difftime(endDate, startDate, unit="mins")))
}

## Extracts the bootnodes that where provided to the crawler
LoadBootNodesFromFile = function(filepath) {
  lines = readLines(filepath, n=-1)
  # Skip the comments
  lines = lines[!grepl("//", lines, fixed=T)]
  nodes = unname(sapply(lines, function(l) {
    return(strsplit(l, "ipfs/")[[1]][2])
  }))
  return(unique(nodes))
}

## LoadDT loads a single .csv or .json into a data.table. By default it assumes there's no header.
LoadDT = function(fullpath, header=F) {
  ## Test if it's a .json or a .csv
  if (grepl(".csv", fullpath, fixed = T)) {
    return(data.table(read.csv(fullpath, header = header, stringsAsFactors = F, sep=";")))
  } else {
    ## We skip the crawl metadata by returning the node array directly.
    return(data.table(fromJSON(fullpath)[[jsonNodeContentIndex]]))
  }
}

## Load one graph from .csv into memory
loadGraph = function(fullpath, online) {
  dt = data.table(read.csv(fullpath, header = T, stringsAsFactors = F, sep=";"))
  if (online == T) {
    dt = dt[ONLINE == "true"]
  }
  dt$ONLINE=NULL
  g = graph_from_edgelist(as.matrix(dt), directed = T)
  return(g)
}

## Confidence Interval calculation using a student t-test => We assume normally distributed values.
CI = function(data, conf.level = 0.95) {
  # Returns (lower, upper)
  # Check if all data entries are equal -> No confidence interval
  if(all(data == data[1])) {
    return(c(data[1], data[1]))
  }
  
  t = t.test(data, conf.level = conf.level)$conf.int
  return(c(t[1], t[2]))
}

## RMFromEval removes all occurences of the specified variable from the .lua output file
RMFromEval = function(var) {
  # The .lua file will be small, so reading it entirely to memory should be no big deal
  luaFile=file(outFilePath,open="rt+wt")
  lines=readLines(luaFile)
  close(luaFile)
  
  # Just those R things: grepl looks if var is contained in each entry of the vector
  # "lines", thus outputting a boolean vector. We then use this boolean vector as an
  # input to lines itself to get all the elements that *do not* contain the variable var.
  lines = lines[!grepl(var, lines, fixed=T)]
  
  # Write this to file
  cat(lines, file=outFilePath, sep="\n", append=F)
}

## GetFromEval returns the value to the specified key from the Eval-File
GetFromEval = function(var) {
  # Again, the file is small, so we can just read to the memory
  luaFile=file(outFilePath,open="rt+wt")
  lines=readLines(luaFile)
  close(luaFile)
  
  ## Same as in RMFromEval: We grep through all the lines and only return those that match
  ## the variable name
  lines = lines[grepl(var, lines, fixed=T)]
  val=as.numeric(strsplit(lines, "=")[[1]][2])
  return(val)
}
