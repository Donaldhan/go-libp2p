package autorelay

import (
	"time"

	"github.com/benbjohnson/clock"
)

type timer struct {
	timer   *clock.Timer //定时器时间
	running bool         //是否运行
	read    bool         //
}

// 创建定时器
func newTimer(cl clock.Clock) *timer {
	t := cl.Timer(100 * time.Hour) // There's no way to initialize a stopped timer
	t.Stop()
	return &timer{timer: t}
}

// 定时器时间
func (t *timer) Chan() <-chan time.Time {
	return t.timer.C
}

// 停止
func (t *timer) Stop() {
	if !t.running {
		return
	}
	if !t.timer.Stop() && !t.read {
		<-t.timer.C
	}
	t.read = false
}

// 设置可读
func (t *timer) SetRead() {
	t.read = true
}

// 重置定时器，可以重新运行
func (t *timer) Reset(d time.Duration) {
	t.Stop()
	t.timer.Reset(d)
}
