package internal

import (
	"log"
	"os"
	"time"
)

func InitLog() {
	fd, err := os.OpenFile("./cube.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(fd)
	log.SetFlags(log.Lmsgprefix) // 去除日志每行开头自带的时间戳前缀
}

func LogWithError(err error, worker *Worker) {
	log.Println(append(append([]interface{}{"\033[0;31m" + time.Now().Format("2006-01-02 15:04:05.000"), worker.Id(), "Error"}, err), "\033[m")...)
}
