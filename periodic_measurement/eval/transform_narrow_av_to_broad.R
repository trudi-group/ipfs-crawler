library(data.table)
library(reshape2)

dt = data.table(read.csv("plot_data/agent_versions_truncated.csv", header=T, stringsAsFactors=F, sep=";"))
setnames(dt, 1:3, c("date", "version", "count"))

newDT = dcast(dt, date ~ version, value.var="count")
write.csv(newDT, "agent_versions_broad.csv", row.names=F)
