package api

import (
	"context"
	"fmt"
	"os"

	"github.com/namesarnav/netmeta/internal/config"
	"github.com/namesarnav/netmeta/pkg/auto"
	"github.com/namesarnav/netmeta/pkg/bgp"
	"github.com/namesarnav/netmeta/pkg/mpls"
	"github.com/namesarnav/netmeta/pkg/monitor"
	"github.com/namesarnav/netmeta/pkg/ospf"
	"github.com/namesarnav/netmeta/pkg/ui"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bgpMonitor    *bgp.Monitor
	ospfParser    *ospf.Parser
	mplsValidator *mpls.Validator
	autoEngine    *auto.Engine
	uiServer      *ui.Server
	exporter      *monitor.Exporter
)

func Initialize(cfg *config.Config) error {
	var err error

	// Initialize BGP monitor
	bgpMonitor, err = bgp.NewMonitor()
	if err != nil {
		return fmt.Errorf("failed to initialize BGP monitor: %w", err)
	}

	// Add configured peers
	for _, peer := range cfg.BGP.Peers {
		port := peer.Port
		if port == 0 {
			port = 179
		}
		if err := bgpMonitor.AddPeer(peer.Address, peer.ASN, port); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add BGP peer %s: %v\n", peer.Address, err)
		}
	}

	// Initialize OSPF parser
	ospfParser = ospf.NewParser()
	if cfg.OSPF.PCAPFile != "" {
		if err := ospfParser.ParsePCAP(cfg.OSPF.PCAPFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse OSPF PCAP: %v\n", err)
		}
	} else if cfg.OSPF.Interface != "" {
		if err := ospfParser.StartLiveCapture(cfg.OSPF.Interface); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to start live OSPF capture: %v\n", err)
		}
	}

	// Initialize MPLS validator
	mplsValidator = mpls.NewValidator()

	// Initialize auto-remediation engine
	autoEngine = auto.NewEngine(cfg, bgpMonitor)

	// Start auto-remediation engine
	ctx := context.Background()
	go autoEngine.Start(ctx)

	// Initialize Prometheus exporter
	exporter = monitor.NewExporter(bgpMonitor, mplsValidator, autoEngine)
	exporter.Start()

	// Initialize UI server
	uiServer = ui.NewServer(cfg, bgpMonitor, ospfParser, autoEngine)

	return nil
}

func Serve(cfg *config.Config) error {
	if err := Initialize(cfg); err != nil {
		return err
	}

	// Add Prometheus metrics endpoint to UI server
	uiServer.GetRouter().GET("/metrics", promhttp.Handler())

	return uiServer.Start()
}

func ListBGPPeers(cfg *config.Config) {
	if bgpMonitor == nil {
		if err := Initialize(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
			return
		}
	}

	peers := bgpMonitor.GetAllPeers()
	fmt.Println("BGP Peers:")
	fmt.Println("Address\t\tASN\tState\t\tPrefixes\tFlaps")
	fmt.Println("------------------------------------------------------------")
	for _, peer := range peers {
		fmt.Printf("%s\t%d\t%s\t%d\t\t%d\n",
			peer.Address, peer.ASN, peer.State, peer.PrefixCount, peer.FlapCount)
	}
}

func ShowOSPFTopology(cfg *config.Config) {
	if ospfParser == nil {
		if err := Initialize(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
			return
		}
	}

	topology := ospfParser.GetTopology()
	fmt.Println("OSPF Topology:")
	fmt.Println("Router ID\tLinks")
	fmt.Println("------------------------------------------------------------")
	for routerID, links := range topology.Routers {
		fmt.Printf("%d\t\t%d links\n", routerID, len(links))
		for _, link := range links {
			fmt.Printf("  -> %d (cost: %d, state: %s)\n",
				link.RemoteRouterID, link.Cost, link.State)
		}
	}
}

func Remediate(cfg *config.Config, peer, prefix, reason string) {
	if autoEngine == nil {
		if err := Initialize(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
			return
		}
	}

	if err := autoEngine.RemediateManual(peer, prefix, reason); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Remediation triggered: peer=%s, prefix=%s, reason=%s\n", peer, prefix, reason)
}

