// +build ignore

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var running bool

func main() {
	running = true
	conn, err := net.DialTimeout("tcp", os.Args[1]+":44444", 3*time.Second)
	if err != nil {
		fmt.Println("connect error")
		return
	}
	defer conn.Close()

	fmt.Printf("Please input your username:")
	r := bufio.NewReader(os.Stdin)
	bufUsername, _, _ := r.ReadLine()
	username := string(bufUsername)
	conn.Write([]byte(username + "\r\n"))
	fmt.Printf("Please input your password:")
	bufPassword, _, _ := r.ReadLine()
	password := string(bufPassword)
	conn.Write([]byte(password + "\r\n"))

	go clientreceiver(conn)

	for running {
		time.Sleep(1 * time.Second)
	}
}

func clientreceiver(conn net.Conn) {
	for {
		buf, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			conn.Close()
			running = false
			fmt.Println("error read")
			break
		}
		fmt.Println(string(buf))
		if string(buf)[:2] != "1:" {
			continue
		}
		cid := strings.Split(string(buf), ":")[1]
		command := strings.Split(string(buf), ":")[2]
		go execCommand(conn, cid, command)
	}
}

func execCommand(conn net.Conn, cid string, command string) {
	commandArgs := strings.Fields(command)

	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		conn.Write([]byte("2:" + cid + ":" + err.Error() + "\r\n"))
		return
	}

	println(out.String())
	result := strings.Replace(out.String(), "\n", "^^n", -1)
	result = strings.Replace(result, ":", "^^", -1)
	conn.Write([]byte("2:" + cid + ":" + result + "\r\n"))
}
