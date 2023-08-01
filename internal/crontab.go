package internal

import (
	"github.com/robfig/cron/v3"
)

var Crontab *cron.Cron // 定时任务

func RunCrontabs(name string) {
	if Crontab == nil { // 首次执行时，先初始化 Crontab
		Crontab = cron.New()
		Crontab.Start()
	}

	if name == "" {
		name = "%"
	}

	rows, err := Db.Query("select name, cron from source where name like ? and type = 'crontab' and active = true", name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n, c string
		rows.Scan(&n, &c)

		if _, ok := Cache.Crontabs[n]; ok { // 防止重复添加任务
			continue
		}

		id, err := Crontab.AddFunc(c, func() {
			worker := <-WorkerPool.Channels
			defer func() {
				WorkerPool.Channels <- worker
			}()

			worker.Run(worker.Runtime().ToValue("./crontab/" + n))
		})
		if err != nil {
			panic(err)
		} else {
			Cache.Crontabs[n] = id
		}
	}
}
