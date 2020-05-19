package crawling

import "time"
import "sync/atomic"
import "fmt"

type stats struct {
  startTime time.Time
  endTime time.Time
  onlineCnt int64
}

func NewStats() stats {
  return stats {
    startTime:  time.Now(),
    endTime:    time.Now(),
    onlineCnt:  0,
  }
}

func (st *stats) Register (cm *CrawlManagerV2)  {
  cm.Events.Subscribe(cm.Events.CRAWL_STARTED, st.logStart)
  cm.Events.Subscribe(cm.Events.CRAWL_ENDED, st.logEnd)
  cm.Events.Subscribe(cm.Events.CON_SUC, st.countOnline)

}

func (st *stats) logStart (string)  {
  st.startTime = time.Now()
}

func (st *stats) logEnd (string)  {
  st.endTime = time.Now()
}

func (st *stats) countOnline (string)  {
  atomic.AddInt64(&st.onlineCnt,1)
}

func (st *stats) Print ()  {
  fmt.Println(st.startTime)
  fmt.Println(st.endTime)
  fmt.Println(st.onlineCnt)
}
