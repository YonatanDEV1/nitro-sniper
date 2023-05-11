package main

import (
	"github.com/sasha-s/go-csync"
	"context"
	"time"
)

type RateLimiter interface {
	Close(ctx context.Context)
	Reset()
	Wait(ctx context.Context) error
	Unlock()
}

func NewRateLimiter(opts ...RateLimiterConfigOpt) RateLimiter {
	config := DefaultRateLimiterConfig()
	config.Apply(opts)

	return &rateLimiterImpl{
		config: *config,
	}
}

type rateLimiterImpl struct {
	mu csync.Mutex

	reset     time.Time
	remaining int

	config RateLimiterConfig
}

func (l *rateLimiterImpl) Close(ctx context.Context) {
	_ = l.mu.CLock(ctx)
}

func (l *rateLimiterImpl) Reset() {
	l.reset = time.Time{}
	l.remaining = 0
	l.mu = csync.Mutex{}
}

func (l *rateLimiterImpl) Wait(ctx context.Context) error {
	if err := l.mu.CLock(ctx); err != nil {
		return err
	}

	now := time.Now()

	var until time.Time

	if l.remaining == 0 && l.reset.After(now) {
		until = l.reset
	}

	if until.After(now) {
		select {
		case <-ctx.Done():
			l.Unlock()
			return ctx.Err()
		case <-time.After(until.Sub(now)):
		}
	}
	return nil
}

func (l *rateLimiterImpl) Unlock() {
	now := time.Now()
	if l.reset.Before(now) {
		l.reset = now.Add(time.Minute)
		l.remaining = l.config.CommandsPerMinute
	}
	l.mu.Unlock()
}

func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		CommandsPerMinute: 120,
	}
}

type RateLimiterConfig struct {
	CommandsPerMinute int
}

type RateLimiterConfigOpt func(config *RateLimiterConfig)

func (c *RateLimiterConfig) Apply(opts []RateLimiterConfigOpt) {
	for _, opt := range opts {
		opt(c)
	}
}

func WithCommandsPerMinute(commandsPerMinute int) RateLimiterConfigOpt {
	return func(config *RateLimiterConfig) {
		config.CommandsPerMinute = commandsPerMinute
	}
}