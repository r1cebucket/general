package main

import (
	"os"
	"os/signal"

	"tcpclient/client"
)

func main() {
	// name := "tmpname"
	// passwd := "tmppasswd"
	// for i := 0; i < 20000; i++ {
	// 	index := strconv.Itoa(i)
	// 	c := client.Client{name + index, passwd + index}
	// 	go c.Start()
	// }
	c0 := client.Client{"tmpname0", "tmppasswd0"}
	go c0.Start()
	// c1 := client.Client{"tmpname1", "tmppasswd1"}
	// go c1.Start()
	// c2 := client.Client{"tmpname2", "tmppasswd2"}
	// go c2.Start()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, os.Interrupt)
	<-quitChan
}
