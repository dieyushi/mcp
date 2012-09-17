package main

import (
	"bytes"
	"code.google.com/p/goconf/conf"
	"flag"
	"fmt"
	"github.com/monnand/goredis"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	redisClient goredis.Client
	mainChan    = make(chan bool)
	closeFdChan = make(chan bool)
	host        string
	webport     string
	pcport      string
)

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	closeFds := flag.Bool("close-fds", false, "Whether or not to close stdin, stdout and stderr")
	call := flag.String("call", "",
		"Call the specified command:"+
			"\n\t\tquit:         send a quit signal to *addr* (equivalent to the GET request: http://*addr*/?data=\"bye ni\")"+
			"\n\t\treplace:      send a quit signal to *addr* then startup as normal"+
			"")
	flag.Parse()

	handleConfig()

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

	println("trying to listen on port", webport, "and", pcport)

	if *d {
		cmd := exec.Command(os.Args[0],
			"-close-fds",
			"-call", *call)

		serr, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatalln(err)
		}
		err = cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}
		s, err := ioutil.ReadAll(serr)
		s = bytes.TrimSpace(s)

		if strings.Contains(string(s), "listen error on port") {
			fmt.Printf("%v\n", string(s))
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

		<-closeFdChan
		<-closeFdChan

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
	if resp, err := http.Get("http://127.0.0.1:" + webport + "/bye/"); err == nil {
		resp.Body.Close()
	}
}

func handleConfig() {
	mcpConfig, err := conf.ReadConfigFile("mcp.conf")
	if err != nil {
		fmt.Printf("parse config error, start up with default config\n")
		host = ""
		webport = "8080"
		pcport = "44444"
		return
	}
	host, _ = mcpConfig.GetString("default", "host")
	webport, _ = mcpConfig.GetString("default", "webport")
	pcport, _ = mcpConfig.GetString("default", "pcport")
}
