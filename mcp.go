package main

import (
	"bytes"
	"flag"
	"github.com/monnand/goredis"
	"io"
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

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			LogE(err)
		}
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}

	switch *call {
	case "":
	case "quit":
		sendQuit()
		return
	case "replace":
	default:
		LogE("invalid call: expected one of `quit, replace', got `%s'\n", *call)
	}

	if *d {
		cmd := exec.Command(os.Args[0], "-close-fds", "-call", *call)

		serr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalln(err)
		}
		err = cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}
		s, err := ioutil.ReadAll(serr)
		s = bytes.TrimSpace(s)

		if strings.Contains(string(s), "listen error on port") ||
			strings.Contains(string(s), "Connect redis error") {
			println(string(s))
			cmd.Process.Kill()
		} else {
			cmd.Process.Release()
			Log("listening on port", webport, "and", pcport)
			Log("Serving in the background")
		}
	} else {
		if *call == "replace" {
			sendQuit()
			cmd := exec.Command(os.Args[0], "-d")
			cmd.Start()
			cmd.Process.Release()
			return
		}

		runtime.GOMAXPROCS(runtime.NumCPU() + 1)

		redisClient.Addr = redisaddr
		redisClient.Db = redisdb
		redisClient.Password = redispwd

		pong, err := redisClient.Ping()
		if err != nil || pong != "PONG" {
			LogE("Connect redis error,exit")
			return
		}

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
	if host == "" {
		if resp, err := http.Get("http://127.0.0.1:" + webport + "/bye/"); err == nil {
			resp.Body.Close()
		}
	} else {
		if resp, err := http.Get("http://" + host + ":" + webport + "/bye/"); err == nil {
			resp.Body.Close()
		}
	}
}
