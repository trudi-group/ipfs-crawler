package crawling

import "time"
import "sync/atomic"
import "fmt"

type stats struct {
  startTime time.Time
  endTime time.Time
  onlineCnt int64
  tryCnt int64
  failCnt int64

}

func NewStats() stats {
  return stats {
    startTime:  time.Now(),
    endTime:    time.Now(),
    onlineCnt:  0,
    tryCnt:     0,
    failCnt:    0,
  }
}

func (st *stats) Register (cm *CrawlManagerV2)  {
  cm.Events.Subscribe(cm.Events.CRAWL_STARTED, st.logStart)
  cm.Events.Subscribe(cm.Events.CRAWL_STARTED, st.debug)
  cm.Events.Subscribe(cm.Events.CRAWL_ENDED, st.logEnd)
  cm.Events.Subscribe(cm.Events.CON_SUC, st.countOnline)
  cm.Events.Subscribe(cm.Events.CON_FAIL, st.countFail)
  cm.Events.Subscribe(cm.Events.CON_TRY, st.countTry)
}

func (st *stats) logStart (string) error {
  st.startTime = time.Now()
  return nil
}

func (st *stats) logEnd (string) error {
  st.endTime = time.Now()
  return nil
}

func (st *stats) countOnline (string) error {
  atomic.AddInt64(&st.onlineCnt,1)
  return nil
}
func (st *stats) countTry (string) error {
  atomic.AddInt64(&st.tryCnt,1)
  return nil
}
func (st *stats) countFail (string) error {
  atomic.AddInt64(&st.failCnt,1)
  return nil
}

func (st *stats) debug (string) error {
  fmt.Println("HelloWorld")
  return nil
}

func (st *stats) Print ()  {
  fmt.Println(st.startTime)
  fmt.Println(st.endTime)
  fmt.Println(st.tryCnt)
  fmt.Println(st.onlineCnt)
  fmt.Println(st.failCnt)

}
