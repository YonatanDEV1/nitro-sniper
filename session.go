package main

import (
	"github.com/gorilla/websocket"
	"time"
	"sync"
)

func socketConnection(token string) (session *Session) {
	s := &Session{
		dialer:   websocket.DefaultDialer,
		sequence: new(int64),
		token:    token,
		gateway:  "wss://gateway.discord.gg",
	}

	if s.RateLimiter == nil {
		s.RateLimiter = NewRateLimiter(s.RateRateLimiterConfigOpts...)
	}

	return s
}

type Session struct {
	RateRateLimiterConfigOpts []RateLimiterConfigOpt
	socketConnection *websocket.Conn
	heartbeatInterval time.Duration
	listening chan interface{}
	dialer *websocket.Dialer
	heartbeatSent time.Time
	RateLimiter RateLimiter
	socketMutex sync.Mutex
	heartbeatAck time.Time
	isConnected bool
	sessionId string
	sequence *int64
	gateway string
	account string
	token string
	sync.RWMutex
}