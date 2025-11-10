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

type KillEvent struct {
	BaseEvent
	AttackerXUID      string
	AttackerClientNum int
	AttackerTeam      string
	AttackerName      string
	VictimXUID        string
	VictimClientNum   int
	VictimTeam        string
	VictimName        string
	Weapon            string
	Damage            string
	MeansOfDeath      string
	HitLocation       string
}

func (b *BaseEvent) GetCommand() string           { return b.Command }
func (b *BaseEvent) GetTimestamp() *time.Duration { return b.Timestamp }
func (b *BaseEvent) GetRaw() string               { return b.Raw }
