package main

import (
	"code.google.com/p/goconf/conf"
)

var (
	host      string
	webport   string
	pcport    string
	webserver bool
	pcserver  bool
	usessl    bool
	redisaddr string
	redisdb   int
	redispwd  string
	weblog    bool
	serlog    bool
	logfile   string
)

func handleConfig() {
	mcpConfig, err := conf.ReadConfigFile("mcp.conf")
	if err != nil {
		Log("parse config error (mcp.conf not found), start up with default config")
		host = ""
		webport = "8080"
		pcport = "44444"
		webserver = true
		pcserver = true
		usessl = true
		redisaddr = ":6379"
		redisdb = 0
		redispwd = ""
		weblog = true
		serlog = true
		logfile = "mcp.log"
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
	webserver, err = mcpConfig.GetBool("default", "webserver")
	if err != nil {
		webserver = true
	}
	pcserver, err = mcpConfig.GetBool("default", "pcserver")
	if err != nil {
		pcserver = true
	}
	usessl, err = mcpConfig.GetBool("default", "usessl")
	if err != nil {
		usessl = true
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
	logfile, err = mcpConfig.GetString("log", "logfile")
	if err != nil {
		logfile = "mcp.log"
	}
}
