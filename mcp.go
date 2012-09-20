package main

import (
	"bytes"
	"code.google.com/p/goconf/conf"
	"flag"
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
	redisaddr   string
	redisdb     int
	redispwd    string
	weblog      bool
	serlog      bool
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
		LogE("invalid call: expected one of `quit, replace', got `%s'\n", *call)
	}

	if *d {
		cmd := exec.Command(os.Args[0],
			"-close-fds",
			"-call", *call)

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
		Log("listening on port", webport, "and", pcport)
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

func handleConfig() {
	mcpConfig, err := conf.ReadConfigFile("mcp.conf")
	if err != nil {
		Log("parse config error (mcp.conf not found), start up with default config")
		host = ""
		webport = "8080"
		pcport = "44444"
		redisaddr = ":6379"
		redisdb = 0
		redispwd = ""
		weblog = true
		serlog = true

		return
	}

	host, err = mcpConfig.GetString("default", "host")
	if err != nil {
		host = ""
	}
	webport, err = mcpConfig.GetString("default", "webport")
	if err != nil {
		webport = "8080"
	}
	pcport, err = mcpConfig.GetString("default", "pcport")
	if err != nil {
		pcport = "44444"
	}
	redisaddr, err = mcpConfig.GetString("redis", "redisaddr")
	if err != nil {
		redisaddr = ":6379"
	}
	redisdb, err = mcpConfig.GetInt("redis", "redisdb")
	if err != nil {
		redisdb = 0
	}
	redispwd, err = mcpConfig.GetString("redis", "redispwd")
	if err != nil {
		redispwd = ""
	}
	weblog, err = mcpConfig.GetBool("log", "weblog")
	if err != nil {
		weblog = true
	}
	serlog, err = mcpConfig.GetBool("log", "serlog")
	if err != nil {
		serlog = true
	}
}

func LogS(v ...interface{}) {
	if serlog {
		log.SetPrefix("[SER] ")
		log.Println(v...)
	}
}

func LogW(v ...interface{}) {
	if weblog {
		log.SetPrefix("[WEB] ")
		log.Println(v...)
	}
}

func Log(v ...interface{}) {
	log.SetPrefix("[MCP] ")
	log.Println(v...)
}

func LogE(v ...interface{}) {
	log.SetPrefix("[MCP] ")
	log.Fatalln(v...)
}
