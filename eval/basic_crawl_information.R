### Basic Crawl Information ###
### Outputs basic crawl information to eval.lua.
### 1) The crawl period
### 2) average crawl duration
### 3) total number of crawls

# Load includes, as always
source("includes.R")

# Get a filelist. The filenames carry important information
crawls = list.files(path=crawlDir, pattern=visitedPattern)

# Extract the *start* date of the crawl
timestamps = pbsapply(crawls, extractStartDate, USE.NAMES = F)
# Somehow the Posix timestamp breaks with sapply and gets converted to numeric
timestamps = as.POSIXct(timestamps, tz = "", origin="1970-01-01")

# Get the durations as well
durations = pbsapply(crawls, crawlDurationFromFilename)

# The duration is simply from the max. to the min. timestamp.
# Technically, we are neglecting the duration of the last crawl, but since the durations
# are in the order of minutes and the crawl period is multiple days long, this is an ok approximation.
crawlPeriod = as.double(max(timestamps) - min(timestamps), unit="days")
numCrawls = length(crawls)

# Get start and end dates
startDate = min(timestamps)
endDate = max(timestamps)

# Write information to eval.lua.
# CrawlPeriod is in days
writeToEvalRounded("crawlPeriod", crawlPeriod)
# Integer
writeToEvalRounded("numCrawls", numCrawls)
# avgCrawlDuration is in minutes.
writeToEvalRounded("avgCrawlDuration", mean(durations))
# Write dates
writeToEval("crawlStartDate", paste("\"",startDate, "\"", sep=""))
writeToEval("crawlEndDate", paste("\"",endDate, "\"", sep=""))
