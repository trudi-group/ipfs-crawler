## num_nodes_plot.R: Plots the number of nodes over time.

# It is called from "eval/"
source("includes.R")

############################### CONSTANTS ########################

dataFile = "num_nodes.csv"
plotDateBreaks = "24 hours"
pointSeparation = 100

############################### PLOTTING #########################

meltedDT = data.table(read.table(
  paste("plot_data/", dataFile, sep="")
    , header=T, sep=";", stringsAsFactors=F))

meltedDT$ts = as.POSIXct(meltedDT$ts)

q = ggplot(meltedDT, aes(x=ts, y=value, color=variable)) + 
  geom_point(data=meltedDT[seq(1, nrow(meltedDT), by=pointSeparation)], aes(shape=variable)) + 
  geom_line(aes(linetype=variable)) +
  xlab("Timestamp") + ylab("Number of nodes") +
  scale_color_discrete(name="Node type", labels=c("All", "Reachable")) +
  scale_linetype_discrete(name="Node type", labels=c("All", "Reachable")) +
  scale_shape_discrete(name="Node type", labels=c("All", "Reachable")) +
  scale_y_continuous(breaks = scales::pretty_breaks(n = plotBreakNumber)) +
  # scale_y_continuous(breaks = seq(0, maxYValue, by=yAxisBreaks)) +
  scale_x_datetime(breaks = plotDateBreaks, date_labels = plotDateFormat) +
  theme(axis.text.x = element_text(angle = 45, vjust = 1, hjust=1))

## Output to .tex but also to .png
# tikz(file=paste(outPlotPath, "num_nodes.tex", sep=""), width=plotWidth, height=plotHeight)
# q
# dev.off()

# pdf(file=paste(outPlotPath, "num_nodes.pdf", sep=""), width=plotWidth, height=plotHeight)
# q
# dev.off()

png(filename=paste(outPlotPath, "num_nodes.png", sep=""), height = bitmapHeight, width=bitmapWidth)
q
dev.off()
