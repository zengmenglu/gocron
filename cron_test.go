package gocron

import (
	"fmt"
	"testing"
	"time"
)

// 测试每10ms触发周期性定时任务
func Test10ms(t *testing.T) {
	c := New()
	c.AddJob(-1, "@every 10ms", func() {
		fmt.Println(time.Now(), "hello world")

	})
	c.Start()
	defer c.Stop()

	select {
	case <-time.After(100 * time.Millisecond):
	}
}

// 测试每个月23日16点40分55秒270毫秒触发周期性定时任务
func TestEveryMonth(t *testing.T) {
	c := New()
	c.AddJob(-1, "* 23 16 43 55 270", func() {
		fmt.Println(time.Now(), "ms:", time.Now().Nanosecond()/1e6, "hello world")
	})
	c.Start()
	defer c.Stop()

	select {}
}

// 测试添加任务
func TestCron_AddJob(t *testing.T) {
	c := New()

	c.AddJob(-1, "@every 1s", func() {
		fmt.Println(time.Now(), "hello world")
	})

	c.AddJob(-1, "@every 3s", func() {
		fmt.Println(time.Now(), "hello world again")
	})

	c.Start()
	defer c.Stop()

	select {
	case <-time.After(time.Minute):
	}
}

// 测试删除任务
func TestCron_RemoveJob(t *testing.T) {
	c := New()
	id, _ := c.AddJob(-1, "@every 1s", func() {
		fmt.Println(time.Now(), "hello world")
	})

	c.Start()
	defer c.Stop()

	go func() {
		time.Sleep(5 * time.Second)
		c.RemoveJob(id)
	}()

	select {
	case <-time.After(10 * time.Minute):

	}
}

// 测试非周期任务
func Test10msOnce(t *testing.T) {
	c := New()
	c.AddJob(1, "@every 10ms", func() {
		fmt.Println(time.Now(), "hello world")

	})
	c.Start()
	defer c.Stop()

	select {
	case <-time.After(100 * time.Millisecond):
	}
}
