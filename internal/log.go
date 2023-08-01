package internal

import (
	"log"
	"os"
)

func InitLog() {
	fd, err := os.OpenFile("./cube.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(fd)
}
