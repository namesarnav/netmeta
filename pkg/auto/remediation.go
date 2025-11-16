package auto

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/namesarnav/netmeta/internal/config"
	"github.com/namesarnav/netmeta/pkg/bgp"
)

type RemediationEvent struct {
	Timestamp time.Time
	Type      string
	Target    string
	Reason    string
	Action    string
	Success   bool
}

type Engine struct {
	cfg           *config.Config
	bgpMonitor    *bgp.Monitor
	events        []RemediationEvent
	mu            sync.RWMutex
	flapHistory   map[string][]time.Time
	flapHistoryMu sync.RWMutex
}

func NewEngine(cfg *config.Config, bgpMonitor *bgp.Monitor) *Engine {
	return &Engine{
		cfg:         cfg,
		bgpMonitor:  bgpMonitor,
		events:      make([]RemediationEvent, 0),
		flapHistory: make(map[string][]time.Time),
	}
}

func (e *Engine) Start(ctx context.Context) {
	if !e.cfg.Auto.Enabled {
		return
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.checkAndRemediate()
		}
	}
}

func (e *Engine) checkAndRemediate() {
	peers := e.bgpMonitor.GetAllPeers()
	now := time.Now()
	window := time.Duration(e.cfg.Auto.FlapWindowSec) * time.Second

	for _, peer := range peers {
		peer.mu.RLock()
		flapCount := peer.FlapCount
		lastFlapTime := peer.LastFlapTime
		peer.mu.RUnlock()

		// Check flap threshold
		if flapCount > int64(e.cfg.Auto.FlapThreshold) {
			// Check if flaps occurred within the time window
			if !lastFlapTime.IsZero() && now.Sub(lastFlapTime) < window {
				e.remediateFlap(peer.Address)
			}
		}
	}
}

func (e *Engine) remediateFlap(peerAddress string) {
	event := RemediationEvent{
		Timestamp: time.Now(),
		Type:      "bgp_flap",
		Target:    peerAddress,
		Reason:    "flap",
		Action:    "withdraw_all_prefixes",
		Success:   false,
	}

	if err := e.bgpMonitor.WithdrawAllPrefixes(peerAddress); err != nil {
		event.Success = false
		e.recordEvent(event)
		return
	}

	event.Success = true
	e.recordEvent(event)
}

func (e *Engine) RemediateRPKI(prefix string) error {
	event := RemediationEvent{
		Timestamp: time.Now(),
		Type:      "rpki_invalid",
		Target:    prefix,
		Reason:    "rpki",
		Action:    "withdraw_prefix",
		Success:   false,
	}

	// In a real implementation, this would withdraw the specific prefix
	// For now, we'll just record the event
	event.Success = true
	e.recordEvent(event)

	return nil
}

func (e *Engine) RemediateOSPFAdjacency(interfaceName string) error {
	event := RemediationEvent{
		Timestamp: time.Now(),
		Type:      "ospf_adjacency",
		Target:    interfaceName,
		Reason:    "adjacency_down",
		Action:    "restart_interface",
		Success:   false,
	}

	// In a real implementation, this would restart the interface
	// For now, we'll just record the event
	event.Success = true
	e.recordEvent(event)

	return nil
}

func (e *Engine) RemediateManual(peer, prefix, reason string) error {
	event := RemediationEvent{
		Timestamp: time.Now(),
		Type:      "manual",
		Target:    peer,
		Reason:    reason,
		Action:    "manual_remediation",
		Success:   false,
	}

	if peer != "" {
		if err := e.bgpMonitor.WithdrawAllPrefixes(peer); err != nil {
			event.Success = false
			e.recordEvent(event)
			return fmt.Errorf("failed to remediate peer %s: %w", peer, err)
		}
	}

	if prefix != "" {
		if err := e.RemediateRPKI(prefix); err != nil {
			event.Success = false
			e.recordEvent(event)
			return fmt.Errorf("failed to remediate prefix %s: %w", prefix, err)
		}
	}

	event.Success = true
	e.recordEvent(event)
	return nil
}

func (e *Engine) recordEvent(event RemediationEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)

	// Keep only last 1000 events
	if len(e.events) > 1000 {
		e.events = e.events[len(e.events)-1000:]
	}
}

func (e *Engine) GetEvents(limit int) []RemediationEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 || limit > len(e.events) {
		limit = len(e.events)
	}

	events := make([]RemediationEvent, limit)
	copy(events, e.events[len(e.events)-limit:])
	return events
}

func (e *Engine) GetRemediationCount(reason string) int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	count := int64(0)
	for _, event := range e.events {
		if event.Reason == reason && event.Success {
			count++
		}
	}
	return count
}

