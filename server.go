package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

type Client struct {
	ID        string
	IN        chan string
	Quit      chan bool
	Conn      net.Conn
	ClientMap map[string]Client
}

var clientMap = make(map[string]Client)

func servermain() {
	ln, err := net.Listen("tcp", ":44444")
	if err != nil {
		fmt.Println("listen error on port 44444")
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept error")
			continue
		}
		go HandleClient(conn, clientMap)
	}
}

func HandleClient(conn net.Conn, clientMap map[string]Client) {
	bufUsername, _, _ := bufio.NewReader(conn).ReadLine()
	bufPassword, _, _ := bufio.NewReader(conn).ReadLine()

	uid, ok := VerifyPasswd(string(bufUsername), string(bufPassword))
	if ok == false {
		conn.Close()
		return
	}

	if IsUIDOnline(uid) {
		conn.Close()
		return
	}

	newClient := &Client{
		ID:        uid,
		IN:        make(chan string),
		Quit:      make(chan bool),
		Conn:      conn,
		ClientMap: clientMap,
	}

	clientMap[newClient.ID] = *newClient

	go clientSender(newClient)
	go clientReceiver(newClient)
}

func clientSender(client *Client) {
	for {
		select {
		case buf := <-client.IN:
			client.Conn.Write([]byte(buf + "\r\n"))
			break
		case <-client.Quit:
			client.Conn.Close()
			break
		}
	}
}

func clientReceiver(client *Client) {
	for {
		buf, _, err := bufio.NewReader(client.Conn).ReadLine()
		if err != nil {
			break
		}
		if string(buf) == string("/quit") {
			client.Close()
			break
		}
		client.IN <- string(buf)
	}
}

func (client *Client) Close() {
	client.Quit <- true
	client.Conn.Close()
	delete(client.ClientMap, client.ID)
}

func VerifyPasswd(username string, password string) (string, bool) {
	if uid, err := redisClient.Get("user:" + username); err == nil {
		passwordInDB, _ := redisClient.Get("user:" + string(uid) + ":pass")
		if string(passwordInDB) == password {
			return string(uid), true
		}
	}
	return "0", false
}

func IsUIDOnline(uid string) bool {
	_, ok := clientMap[uid]
	return ok
}

func CommResult(uid string, cid string, result string) {
	redisClient.Set("comm:"+cid+":result", []byte(result))
	CommDoneCallback(uid, cid)
}

func CommDoneCallback(uid string, cid string) {
	redisClient.Zrem("comm:"+uid+":donecids", []byte(cid))
	score, _ := strconv.Atoi(cid)
	redisClient.Zadd("comm:"+uid+":donecids", []byte(cid), float64(score))
}

func AddEventFromWeb(uid string, cid string, command string) {
	if IsUIDOnline(uid) == false {
		return
	}

	clientMap[uid].IN <- cid + ":" + command
}
