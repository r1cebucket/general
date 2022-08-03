package main

import (
	"os"
	"os/signal"
	"strconv"

	"tcpclient/client"
)

func main() {
	// testSingle()
	go testMass1(50000, "127.0.0.2")
	go testMass2(50000, "127.0.0.3")

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, os.Interrupt)
	<-quitChan
}

func testSingle() {
	u0 := client.User{"tmpname0", "tmppasswd0"}
	c0 := client.Client{}
	c0.Init(u0, "127.0.0.1")
	go c0.Start()
}

func testMass1(num int, localIP string) {
	name := "tmpname"
	passwd := "tmppasswd"
	for i := 0; i < num; i++ {
		index := strconv.Itoa(i)
		u := client.User{name + index, passwd + index}
		c := client.Client{}
		c.Init(u, localIP)
		go c.Start()
	}
}

func testMass2(num int, localIP string) {
	name := "tmpname"
	passwd := "tmppasswd"
	for i := 50000; i < num+50000; i++ {
		index := strconv.Itoa(i)
		u := client.User{name + index, passwd + index}
		c := client.Client{}
		c.Init(u, localIP)
		go c.Start()
	}
}
