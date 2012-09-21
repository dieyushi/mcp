package main

import (
	"log"
)

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
