package telemetry

import (
	"encoding/json"
	"log"
	"time"
)

type EventType string

const (
	EventTypeBGPFlap        EventType = "bgp_flap"
	EventTypeRPKIInvalid    EventType = "rpki_invalid"
	EventTypeOSPFAdjacency  EventType = "ospf_adjacency"
	EventTypeRemediation    EventType = "remediation"
	EventTypeMPLSCorruption EventType = "mpls_corruption"
)

type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Logger struct {
	events chan Event
}

func NewLogger() *Logger {
	l := &Logger{
		events: make(chan Event, 1000),
	}

	go l.processEvents()
	return l
}

func (l *Logger) LogEvent(eventType EventType, source, message string, metadata map[string]interface{}) {
	event := Event{
		Timestamp: time.Now(),
		Type:      eventType,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	}

	select {
	case l.events <- event:
	default:
		// Channel full, drop event
		log.Printf("Warning: event channel full, dropping event: %s", message)
	}
}

func (l *Logger) processEvents() {
	for event := range l.events {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}
		log.Printf("Event: %s", string(data))
	}
}

func (l *Logger) Close() {
	close(l.events)
}

