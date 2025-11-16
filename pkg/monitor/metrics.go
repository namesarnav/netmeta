package monitor

import (
	"github.com/namesarnav/netmeta/pkg/auto"
	"github.com/namesarnav/netmeta/pkg/bgp"
	"github.com/namesarnav/netmeta/pkg/mpls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BGP metrics
	bgpPeerUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bgp_peer_up",
			Help: "BGP peer up status (1 = up, 0 = down)",
		},
		[]string{"peer"},
	)

	bgpPrefixCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bgp_prefix_count",
			Help: "Number of prefixes advertised by BGP peer",
		},
		[]string{"peer", "afi"},
	)

	bgpSessionFlaps = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bgp_session_flaps_total",
			Help: "Total number of BGP session flaps",
		},
		[]string{"peer"},
	)

	// MPLS metrics
	mplsCorruptionEvents = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mpls_corruption_events_total",
			Help: "Total number of MPLS corruption events",
		},
	)

	// Remediation metrics
	remediationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "netmeta_remediation_total",
			Help: "Total number of remediation actions",
		},
		[]string{"reason", "success"},
	)
)

type Exporter struct {
	bgpMonitor    *bgp.Monitor
	mplsValidator *mpls.Validator
	autoEngine    *auto.Engine
}

func NewExporter(bgpMonitor *bgp.Monitor, mplsValidator *mpls.Validator, autoEngine *auto.Engine) *Exporter {
	return &Exporter{
		bgpMonitor:    bgpMonitor,
		mplsValidator: mplsValidator,
		autoEngine:    autoEngine,
	}
}

func (e *Exporter) UpdateMetrics() {
	// Update BGP metrics
	peers := e.bgpMonitor.GetAllPeers()
	for _, peer := range peers {
		if peer.Established {
			bgpPeerUp.WithLabelValues(peer.Address).Set(1)
		} else {
			bgpPeerUp.WithLabelValues(peer.Address).Set(0)
		}

		bgpPrefixCount.WithLabelValues(peer.Address, "ipv4").Set(float64(peer.PrefixCount))
		bgpSessionFlaps.WithLabelValues(peer.Address).Add(0) // Counter, so we set the value
	}

	// Update MPLS metrics
	corruptionCount := e.mplsValidator.GetCorruptionCount()
	mplsCorruptionEvents.Add(0) // This would need to track deltas

	// Update remediation metrics
	reasons := []string{"flap", "rpki", "adjacency_down", "manual"}
	for _, reason := range reasons {
		count := e.autoEngine.GetRemediationCount(reason)
		remediationTotal.WithLabelValues(reason, "true").Add(0) // Would need delta tracking
		_ = count
		_ = corruptionCount
	}
}

// Start starts the metrics update loop
func (e *Exporter) Start() {
	// Metrics are automatically exported via prometheus registry
	// This method can be used for periodic updates if needed
}

