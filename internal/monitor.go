package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/process"
)

func RunMonitor() {
	p, _ := process.NewProcess(int32(os.Getppid()))
	ticker := time.NewTicker(time.Millisecond * 1000)
	for range ticker.C {
		c, _ := p.CPUPercent()
		m, _ := p.MemoryInfo()
		fmt.Printf("\rcpu: %.2f%%, memory: %.2fmb, vm: %d/%d"+" ", // 结尾预留一个空格防止刷新过程中因字符串变短导致上一次打印的文本在结尾出溢出
			c,
			float32(m.RSS)/1024/1024,
			len(WorkerPool.Workers)-len(WorkerPool.Channels), len(WorkerPool.Workers),
		)
	}
}
