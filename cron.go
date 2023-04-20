package gocron

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// 定时器
type Cron struct {
	entries     map[int]*Entry
	entriesList []*Entry
	nextID      int // 定时任务的ID
	running     bool
	runningMu   sync.Mutex

	jobWaiter sync.WaitGroup // 定时任务计数

	add    chan *Entry
	remove chan int
	stop   chan struct{}
}

// 定时任务
type Entry struct {
	ID       int
	Repeat   int64
	Next     time.Time // 下一次执行该任务的时间
	Prev     time.Time // 上一次执行该任务的时间
	Job      func()
	Schedule Schedule
}

// 定时任务调度
type Schedule interface {
	// Next returns the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

func New() *Cron {
	return &Cron{
		entries:     make(map[int]*Entry),
		entriesList: nil,
		add:         make(chan *Entry),
		remove:      make(chan int),
		stop:        make(chan struct{}),
		running:     false,
	}
}

// AddJob
// 入参：
// repeat：任务重复执行次数， -1 表示为周期任务
// spec: 定时任务周期，@every 或 "month,d,h,min,s,ms"
// f: 定时任务具体操作
//
// 出参：
// 定时任务ID
// error
func (c *Cron) AddJob(repeat int64, spec string, f func()) (int, error) {
	schedule, err := Parse(spec)
	if err != nil {
		return 0, err
	}

	c.runningMu.Lock()
	defer c.runningMu.Unlock()

	c.nextID++
	entry := &Entry{
		ID:       c.nextID,
		Repeat:   repeat,
		Schedule: schedule,
		Job:      f,
	}

	if !c.running {
		c.entries[entry.ID] = entry
		c.entriesList = append(c.entriesList, entry)
	} else {
		c.add <- entry
	}
	return entry.ID, nil
}

func (c *Cron) RemoveJob(id int) {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()

	if c.running {
		c.remove <- id
	} else {
		delete(c.entries, id)
	}
}

func (c *Cron) Start() {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()

	if c.running {
		return
	}
	c.running = true
	go c.run()
}

// byTime is a wrapper for sorting the entry array by time
// (with zero time at the end).
type byTime []*Entry

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}
	return s[i].Next.Before(s[j].Next)
}

// run the scheduler.. this is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run() {
	fmt.Println("start")

	// Figure out the next activation times for each entry.
	now := time.Now()
	for _, entry := range c.entries {
		entry.Next = entry.Schedule.Next(now)
		fmt.Println("schedule", "now", now, "entry", entry.ID, "next", entry.Next)
	}

	for {
		// 删除失效任务
		c.delInValidEntry()

		// Determine the next entry to run.
		sort.Sort(byTime(c.entriesList))

		var timer *time.Timer
		if len(c.entries) == 0 || len(c.entriesList) == 0 || c.entriesList[0].Next.IsZero() {
			// If there are no entries yet, just sleep - it still handles new entries
			// and stop requests.
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(c.entriesList[0].Next.Sub(now))
		}

		for {
			select {
			case now = <-timer.C:
				now = time.Now()
				//fmt.Println("wake", "now", now)

				// Run every entry whose next time was less than now
				for _, e := range c.entriesList {
					if e.Next.After(now) || e.Next.IsZero() {
						break
					}
					if _, ok := c.entries[e.ID]; !ok || e.Repeat == 0 {
						continue
					}

					c.startJob(e.Job)
					e.Prev = e.Next

					// 是否是周期性任务
					if e.Repeat == -1 {
						e.Next = e.Schedule.Next(now)
						//fmt.Println("schedule", "now", now, "entry", e.ID, "next", e.Next)
					} else {
						e.Repeat--
					}

					//fmt.Println("run", "now", now, "entry", e.ID, "next", e.Next)
				}

			case newEntry := <-c.add:
				timer.Stop()
				now = time.Now()
				newEntry.Next = newEntry.Schedule.Next(now)
				c.entriesList = append(c.entriesList, newEntry)
				c.entries[newEntry.ID] = newEntry
				//fmt.Println("added", "now", now, "entry", newEntry.ID, "next", newEntry.Next)

			case id := <-c.remove:
				timer.Stop()
				now = time.Now()
				delete(c.entries, id)
				fmt.Println("removed", "entry", id)

			case <-c.stop:
				timer.Stop()
				fmt.Println("stop")
				return
			}

			break
		}
	}
}

func (c *Cron) delInValidEntry() {
	if len(c.entriesList) == len(c.entries) {
		return
	}

	newEntries := make([]*Entry, 0, len(c.entries))
	for i := 0; i < len(c.entriesList); i++ {
		if entry, ok := c.entries[c.entriesList[i].ID]; ok {
			newEntries = append(newEntries, entry)
		}
	}
	c.entriesList = newEntries
}

// startJob runs the given job in a new goroutine.
func (c *Cron) startJob(j func()) {
	c.jobWaiter.Add(1)
	go func() {
		defer c.jobWaiter.Done()
		j()
	}()
}

// Stop stops the cron scheduler if it is running; otherwise it does nothing.
// A context is returned so the caller can wait for running jobs to complete.
func (c *Cron) Stop() context.Context {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()

	if c.running {
		c.stop <- struct{}{}
		c.running = false
	}

	// 优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.jobWaiter.Wait()
		cancel()
	}()
	return ctx
}
