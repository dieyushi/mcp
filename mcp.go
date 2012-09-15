package main

import (
	"github.com/monnand/goredis"
)

var redisClient goredis.Client

func main() {
	redisClient.Addr = ":6379"
	var c chan int
	go webmain()
	go servermain()
	<-c
}
