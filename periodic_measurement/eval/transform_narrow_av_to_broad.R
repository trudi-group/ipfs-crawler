library(data.table)
library(tidyr)

dt = data.table(read.csv("plot_data/agent_versions_truncated.csv", header=T, stringsAsFactors=F, sep=";"))
setnames(dt, 1:3, c("date", "version", "count"))


wdt = pivot_wider(dt, names_from = "version", values_from = "count")
#newDT = dcast(dt, date ~ version, value.var="count")
write.csv(wdt, "agent_versions_broad.csv", row.names=F)
