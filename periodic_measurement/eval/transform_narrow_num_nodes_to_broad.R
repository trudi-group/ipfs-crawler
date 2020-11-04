library(data.table)
library(tidyr)

dt = data.table(read.csv("plot_data/num_nodes.csv", header=F, stringsAsFactors=F, sep=";"))
setnames(dt, 1:3, c("date", "type", "count"))

wdt = pivot_wider(dt, names_from="type", values_from="count")

write.csv(wdt, "num_nodes_broad.csv", row.names=F)
