package session

import (
	"time"
)

type Event struct {
	id        EventId
	timestamp time.Time
	direction Direction
	method    string
	payload   []byte
	size      int
	riskScore RiskScore
}

type EventId struct {
	value string
}

type Direction int

const (
	Inbound Direction = iota
	Outbound
)

type RiskScore struct {
	value int
	level RiskLevel
}

type RiskLevel int

const (
	Low RiskLevel = iota
	Medium
	High
)

func (e Event) IsHighRisk() bool {
	return e.riskScore.level == High
}

func (e Event) ShouldBatch() bool {
	return e.size < 100*1024
	//return e.direction == Inbound && e.RiskScore().level != High
}
