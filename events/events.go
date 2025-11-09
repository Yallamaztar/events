package events

import "time"

type Event interface {
	GetCommand() string
	GetTimestamp() *time.Duration
	GetRaw() string
}

type BaseEvent struct {
	Timestamp *time.Duration
	Command   string
	Raw       string
}

type PlayerEvent struct {
	BaseEvent
	XUID    string
	Flag    int
	Player  string
	Message string
}

type ServerEvent struct {
	BaseEvent
	Data map[string]string
}

func (b *BaseEvent) GetCommand() string           { return b.Command }
func (b *BaseEvent) GetTimestamp() *time.Duration { return b.Timestamp }
func (b *BaseEvent) GetRaw() string               { return b.Raw }
