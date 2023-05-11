package main

import (
	"os/signal"
	"syscall"
	"context"
	"os"
)


func claimer(token string) {
	var session *Session = socketConnection(token)

	defer session.Close(context.TODO())

	var connectionError error = session.Connect(context.TODO())
	_ = connectionError

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}