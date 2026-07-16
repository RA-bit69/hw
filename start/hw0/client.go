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

func connectToServer() error{
	conn, err := net.Dial("tcp", ":8080")
	defer conn.Close()
	if err != nil {
		return err
	}

	connReader := bufio.NewReader(conn)
	data, _ := connReader.ReadString('\n')
	if data != "OK\n" {
		var connerr error
		connerr = ConnectError("Connection error")
		return connerr
	}
	return nil
}

func main() {
	err := connectToServer()
	if err != nil{
		fmt.Println(err)
		return
	}
}