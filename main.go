package main

import (
	"lab/internal/server"
	"log"
	"os"
	"os/signal"
)

func main() {
	go server.Start()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, os.Interrupt)
	<-quitChan
	log.Println("quit")
}
