package main

import (
	"bytes"
	"flag"
	"github.com/monnand/goredis"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var redisClient goredis.Client
var mainChan = make(chan bool)

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	flag.Parse()

	println("trying to listen on port 8080 and 44444")

	if *d {
		cmd := exec.Command(os.Args[0])
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Start()
		if strings.Contains(out.String(), "error") {
			log.Printf("error: %v", out.String())
			cmd.Process.Kill()
		} else {
			cmd.Process.Release()
			println("Serving in the background")
		}
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU() + 1)
		redisClient.Addr = ":6379"

		go webmain()
		go servermain()

		<-mainChan
		<-mainChan
	}
}
