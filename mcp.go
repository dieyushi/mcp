package main

import (
	"github.com/monnand/goredis"
	"runtime"
)

var redisClient goredis.Client
var mainChan = make(chan bool)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
	redisClient.Addr = ":6379"

	go webmain()
	go servermain()

	<-mainChan
	<-mainChan
}
