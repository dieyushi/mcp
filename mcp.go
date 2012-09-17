package main

import (
	"bytes"
	"flag"
	"github.com/monnand/goredis"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var redisClient goredis.Client
var mainChan = make(chan bool)

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	closeFds := flag.Bool("close-fds", false, "Whether or not to close stdin, stdout and stderr")
	call := flag.String("call", "",
		"Call the specified command:"+
			"\n\t\tquit:         send a quit signal to *addr* (equivalent to the GET request: http://*addr*/?data=\"bye ni\")"+
			"\n\t\treplace:      send a quit signal to *addr* then startup as normal"+
			"")
	flag.Parse()

	switch *call {
	case "":
		// startup as normal
	case "quit":
		sendQuit()
		return
	case "replace":
		// handled below
	default:
		log.Fatalf("invalid call: expected one of `quit, replace', got `%s'\n", *call)
	}

	println("trying to listen on port 8080 and 44444")

	if *d {
		cmd := exec.Command(os.Args[0],
			"-close-fds",
			"-call", *call)
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
		if *call == "replace" {
			sendQuit()
		}

		runtime.GOMAXPROCS(runtime.NumCPU() + 1)
		redisClient.Addr = ":6379"

		go webmain()
		go servermain()

		if *closeFds {
			os.Stdin.Close()
			os.Stdout.Close()
			os.Stderr.Close()
		}

		<-mainChan
		<-mainChan
	}
}

func sendQuit() {
	if resp, err := http.Get("http://127.0.0.1:8080/bye/"); err == nil {
		resp.Body.Close()
	}
}
