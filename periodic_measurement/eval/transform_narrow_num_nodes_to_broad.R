library(data.table)

dt = data.table(read.csv("plot_data/num_nodes.csv", header=F, stringsAsFactors=F, sep=";"))
setnames(dt, 1:3, c("date", "type", "count"))

newDT = data.table(date=dt[seq(1, nrow(dt), 2)]$date, all=dt[type=="all"]$count, reachable=dt[type=="online"]$count)
write.csv(newDT, "num_nodes_broad.csv", row.names=F, sep=",")
