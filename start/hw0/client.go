package main

import (
	"bufio"
	"fmt"
	"net"
)

type ConnectError string

func (c ConnectError) Error() string {
	return string(c)
}

func connectToServer() error {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		return err
	}
	defer conn.Close()

	connReader := bufio.NewReader(conn)
	data, err := connReader.ReadString('\n')
	if err != nil {
		return err
	}

	if data != "OK\n" {
		var connerr error
		connerr = ConnectError("Wrong response")
		return connerr
	}
	return nil
}

func main() {
	err := connectToServer()
	if err != nil {
		fmt.Println(err)
		return
	}
}
