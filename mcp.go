package main

import (
	"github.com/monnand/goredis"
	"runtime"
)

var redisClient goredis.Client

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
	redisClient.Addr = ":6379"
	var c chan int
	go webmain()
	go servermain()
	<-c
}
