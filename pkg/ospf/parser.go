package ospf

import (
	"fmt"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type RouterID uint32
type Link struct {
	RemoteRouterID RouterID
	Cost           uint16
	State          string
}

type Topology struct {
	Routers map[RouterID][]Link
	mu      sync.RWMutex
}

type OSPFPacket struct {
	RouterID      RouterID
	Type          layers.OSPFType
	LinkStateID   uint32
	AdvertisingRouter RouterID
	DR            RouterID
	BDR           RouterID
	Neighbors     []RouterID
}

type Parser struct {
	topology *Topology
	handle   *pcap.Handle
}

func NewParser() *Parser {
	return &Parser{
		topology: &Topology{
			Routers: make(map[RouterID][]Link),
		},
	}
}

func (p *Parser) ParsePCAP(filename string) error {
	handle, err := pcap.OpenOffline(filename)
	if err != nil {
		return fmt.Errorf("error opening pcap file: %w", err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for packet := range packetSource.Packets() {
		ospfLayer := packet.Layer(layers.LayerTypeOSPF)
		if ospfLayer == nil {
			continue
		}

		ospf, ok := ospfLayer.(*layers.OSPF)
		if !ok {
			continue
		}

		p.processOSPFPacket(ospf)
	}

	return nil
}

func (p *Parser) processOSPFPacket(ospf *layers.OSPF) {
	p.topology.mu.Lock()
	defer p.topology.mu.Unlock()

	switch ospf.Type {
	case layers.OSPFHello:
		if hello := ospf.Hello; hello != nil {
			routerID := RouterID(ospf.RouterID)
			
			// Initialize router if not exists
			if _, exists := p.topology.Routers[routerID]; !exists {
				p.topology.Routers[routerID] = []Link{}
			}

			// Extract DR and BDR
			// Note: gopacket's OSPF layer may need additional parsing
			// This is a simplified version
		}

	case layers.OSPFDatabaseDescription:
		// Process DBD packets
		routerID := RouterID(ospf.RouterID)
		if _, exists := p.topology.Routers[routerID]; !exists {
			p.topology.Routers[routerID] = []Link{}
		}

	case layers.OSPFLinkStateRequest:
		// Process LSR packets
		routerID := RouterID(ospf.RouterID)
		if _, exists := p.topology.Routers[routerID]; !exists {
			p.topology.Routers[routerID] = []Link{}
		}

	case layers.OSPFLinkStateUpdate:
		if lsu := ospf.LSU; lsu != nil {
			routerID := RouterID(ospf.RouterID)
			
			// Process LSA updates
			for _, lsa := range lsu.LSAs {
				link := Link{
					RemoteRouterID: RouterID(lsa.AdvertisingRouter),
					Cost:           100, // Default cost
					State:          "Up",
				}

				if _, exists := p.topology.Routers[routerID]; !exists {
					p.topology.Routers[routerID] = []Link{}
				}

				// Add link if not duplicate
				links := p.topology.Routers[routerID]
				found := false
				for _, l := range links {
					if l.RemoteRouterID == link.RemoteRouterID {
						found = true
						break
					}
				}
				if !found {
					p.topology.Routers[routerID] = append(links, link)
				}
			}
		}

	case layers.OSPFLinkStateAcknowledgment:
		// Process LSA ACK
		routerID := RouterID(ospf.RouterID)
		if _, exists := p.topology.Routers[routerID]; !exists {
			p.topology.Routers[routerID] = []Link{}
		}
	}
}

func (p *Parser) GetTopology() *Topology {
	p.topology.mu.RLock()
	defer p.topology.mu.RUnlock()

	// Return a copy
	topo := &Topology{
		Routers: make(map[RouterID][]Link),
	}
	for k, v := range p.topology.Routers {
		links := make([]Link, len(v))
		copy(links, v)
		topo.Routers[k] = links
	}
	return topo
}

func (p *Parser) StartLiveCapture(interfaceName string) error {
	handle, err := pcap.OpenLive(interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("error opening interface: %w", err)
	}

	// Set filter for OSPF
	err = handle.SetBPFFilter("ip proto 89")
	if err != nil {
		return fmt.Errorf("error setting BPF filter: %w", err)
	}

	p.handle = handle

	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range packetSource.Packets() {
			ospfLayer := packet.Layer(layers.LayerTypeOSPF)
			if ospfLayer == nil {
				continue
			}

			ospf, ok := ospfLayer.(*layers.OSPF)
			if !ok {
				continue
			}

			p.processOSPFPacket(ospf)
		}
	}()

	return nil
}

func (p *Parser) Close() {
	if p.handle != nil {
		p.handle.Close()
	}
}

