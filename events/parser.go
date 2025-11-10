package events

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseJoinEvent(line string, ts *time.Duration, raw string) (*PlayerEvent, error) {
	m := regexp.MustCompile(`^(J);(-?[A-Fa-f0-9_]{1,32}|bot[0-9]+|0);([0-9]+);(.*)$`).FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("not a join event")
	}

	cmd := m[1]
	originNet := m[2]
	clientNumStr := m[3]
	originName := m[4]

	clientNum, err := strconv.Atoi(clientNumStr)
	if err != nil {
		return nil, fmt.Errorf("invalid client number %q: %w", clientNumStr, err)
	}

	return &PlayerEvent{
		BaseEvent: BaseEvent{
			Timestamp: ts,
			Command:   cmd,
			Raw:       raw,
		},
		XUID:    originNet,
		Flag:    clientNum,
		Player:  originName,
		Message: "",
	}, nil
}

func parseKillEvent(line string, ts *time.Duration, raw string) (*KillEvent, error) {
	parts := strings.Split(line, ";")
	if len(parts) != 13 {
		return nil, fmt.Errorf("not a kill event - expected 13 fields, got %d", len(parts))
	}

	if parts[0] != "K" {
		return nil, fmt.Errorf("not a kill event")
	}

	killerClientNum, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid killer client number %q: %w", parts[2], err)
	}

	victimClientNum, err := strconv.Atoi(parts[6])
	if err != nil {
		return nil, fmt.Errorf("invalid victim client number %q: %w", parts[6], err)
	}

	return &KillEvent{
		BaseEvent: BaseEvent{
			Timestamp: ts,
			Command:   "K",
			Raw:       raw,
		},
		KillerXUID:      parts[1],
		KillerClientNum: killerClientNum,
		KillerTeam:      parts[3],
		KillerName:      parts[4],
		VictimXUID:      parts[5],
		VictimClientNum: victimClientNum,
		VictimTeam:      parts[7],
		VictimName:      parts[8],
		Weapon:          parts[9],
		Damage:          parts[10],
		MeansOfDeath:    parts[11],
		HitLocation:     parts[12],
	}, nil
}

func ParseEventLine(line string) (Event, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	raw := line
	var ts *time.Duration

	fields := strings.Fields(line)
	if len(fields) > 1 {
		first := fields[0]
		if strings.Contains(first, ":") {
			if dur, err := parseTimestamp(first); err == nil {
				ts = &dur
				line = strings.Join(fields[1:], " ")
			}
		}
	}

	if strings.HasPrefix(line, "InitGame:") {
		data := parseKeyValuePairs(strings.TrimPrefix(line, "InitGame:"))
		return &ServerEvent{
			BaseEvent: BaseEvent{
				Timestamp: ts,
				Command:   "InitGame",
				Raw:       raw,
			},
			Data: data,
		}, nil
	}

	if strings.HasPrefix(line, "ShutdownGame:") {
		return &ServerEvent{
			BaseEvent: BaseEvent{
				Timestamp: ts,
				Command:   "ShutdownGame",
				Raw:       raw,
			},
			Data: map[string]string{},
		}, nil
	}

	if strings.Contains(line, ";") {
		if ev, err := parseJoinEvent(line, ts, raw); err == nil {
			return ev, nil
		}
		if ev, err := parseKillEvent(line, ts, raw); err == nil {
			return ev, nil
		}
		return parsePlayerEvent(line, ts, raw)
	}

	if strings.HasPrefix(line, "say ") || strings.HasPrefix(line, "sayteam ") {
		return parseChatPlayerEvent(line, ts, raw)
	}

	return &BaseEvent{
		Timestamp: ts,
		Command:   line,
		Raw:       raw,
	}, nil
}

func parsePlayerEvent(line string, ts *time.Duration, raw string) (*PlayerEvent, error) {
	parts := strings.SplitN(line, ";", 5)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid player event line: %q", line)
	}

	cmd := strings.TrimSpace(parts[0])
	xuid := strings.TrimSpace(parts[1])

	flag, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("invalid flag %q: %w", parts[2], err)
	}

	player := strings.TrimSpace(parts[3])
	message := ""
	if len(parts) == 5 {
		message = strings.TrimSpace(parts[4])
	}

	return &PlayerEvent{
		BaseEvent: BaseEvent{
			Timestamp: ts,
			Command:   cmd,
			Raw:       raw,
		},
		XUID:    xuid,
		Flag:    flag,
		Player:  player,
		Message: message,
	}, nil
}

func parseChatPlayerEvent(line string, ts *time.Duration, raw string) (*PlayerEvent, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid chat play event line: %q", line)
	}

	cmd := fields[0]
	player := fields[1]
	msg := strings.Join(fields[2:], " ")

	return &PlayerEvent{
		BaseEvent: BaseEvent{
			Timestamp: ts,
			Command:   cmd,
			Raw:       raw,
		},
		Player:  player,
		XUID:    "",
		Message: msg,
		Flag:    0,
	}, nil
}

func parseKeyValuePairs(s string) map[string]string {
	data := make(map[string]string)
	s = strings.TrimSpace(s)
	if s == "" {
		return data
	}
	parts := strings.Split(s, "\\")
	for i := 1; i < len(parts)-1; i += 2 {
		key := parts[i]
		val := parts[i+1]
		data[key] = val
	}
	return data
}

func parseTimestamp(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("invalid timestamp: %q", s)
	}

	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, err
		}
		nums[i] = n
	}

	var totalSec int
	switch len(nums) {
	case 2:
		totalSec = nums[0]*60 + nums[1]
	case 3:
		totalSec = nums[0]*3600 + nums[1]*60 + nums[2]
	}

	return time.Duration(totalSec) * time.Second, nil
}
