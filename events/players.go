package events

import (
	"strings"
	"sync"
	"time"
)

const defaultPlayerCacheExpiry = 2 * time.Second

type Player struct {
	ClientNum int
	Name      string
	GUID      string
}

type PlayerSource interface {
	Status() ([]Player, error)
}

type PlayerDirectory struct {
	source  PlayerSource
	ttl     time.Duration
	mu      sync.RWMutex
	players []Player
	expires time.Time
}

func NewPlayerDirectory(source PlayerSource, ttl time.Duration) *PlayerDirectory {
	if ttl <= 0 {
		ttl = defaultPlayerCacheExpiry
	}
	return &PlayerDirectory{source: source, ttl: ttl}
}

func (d *PlayerDirectory) Snapshot() ([]Player, error) {
	d.mu.RLock()
	if len(d.players) > 0 && time.Now().Before(d.expires) {
		result := make([]Player, len(d.players))
		copy(result, d.players)
		d.mu.RUnlock()
		return result, nil
	}
	d.mu.RUnlock()

	players, err := d.source.Status()
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.players = make([]Player, len(players))
	copy(d.players, players)
	d.expires = time.Now().Add(d.ttl)
	d.mu.Unlock()

	result := make([]Player, len(players))
	copy(result, players)
	return result, nil
}

func (d *PlayerDirectory) FindByName(name string) (*Player, error) {
	name = strings.TrimSpace(stripColorCodes(name))
	if name == "" {
		return nil, nil
	}

	players, err := d.Snapshot()
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(name)
	for _, p := range players {
		candidate := strings.ToLower(stripColorCodes(p.Name))
		if strings.Contains(candidate, lower) {
			player := p
			return &player, nil
		}
	}

	return nil, nil
}

func (d *PlayerDirectory) FindByClientNum(clientNum int) (*Player, error) {
	players, err := d.Snapshot()
	if err != nil {
		return nil, err
	}

	for _, p := range players {
		if p.ClientNum == clientNum {
			player := p
			return &player, nil
		}
	}
	return nil, nil
}

func (d *PlayerDirectory) FindByGUID(guid string) (*Player, error) {
	guid = strings.TrimSpace(guid)
	if guid == "" {
		return nil, nil
	}

	players, err := d.Snapshot()
	if err != nil {
		return nil, err
	}

	for _, p := range players {
		if strings.EqualFold(strings.TrimSpace(p.GUID), guid) {
			player := p
			return &player, nil
		}
	}
	return nil, nil
}

func (d *PlayerDirectory) Invalidate() {
	d.mu.Lock()
	d.players = nil
	d.expires = time.Time{}
	d.mu.Unlock()
}

func stripColorCodes(input string) string {
	if input == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input))

	i := 0
	for i < len(input) {
		if input[i] == '^' {
			i++
			if i < len(input) {
				i++
			}
			continue
		}
		b.WriteByte(input[i])
		i++
	}

	return b.String()
}
