package internal

func RunDaemons(name string) {
	if name == "" {
		name = "%"
	}

	rows, err := Db.Query("select name from source where name like ? and type = 'daemon' and active = true", name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		rows.Scan(&n)

		if Cache.Daemons[n] != nil { // 防止重复执行
			continue
		}

		go func() {
			worker := <-WorkerPool.Channels
			defer func() {
				WorkerPool.Channels <- worker
			}()

			Cache.Daemons[n] = worker

			worker.Run(worker.Runtime().ToValue("./daemon/" + n))

			worker.Runtime().ClearInterrupt()

			delete(Cache.Daemons, n)
		}()
	}
}
