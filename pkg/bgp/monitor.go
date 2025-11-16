package bgp

import (
	"context"
	"fmt"
	"sync"
	"time"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/types/known/anypb"
)

type PeerState struct {
	Address      string
	ASN          uint32
	State        string
	PrefixCount  int64
	FlapCount    int64
	LastFlapTime time.Time
	Established  bool
	mu           sync.RWMutex
}

type Monitor struct {
	server   *server.BgpServer
	peers    map[string]*PeerState
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewMonitor() (*Monitor, error) {
	s := server.NewBgpServer()
	go s.Serve()

	ctx, cancel := context.WithCancel(context.Background())

	m := &Monitor{
		server: s,
		peers:  make(map[string]*PeerState),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start monitoring
	go m.monitorPeers()

	return m, nil
}

func (m *Monitor) AddPeer(address string, asn uint32, port uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer := &PeerState{
		Address:     address,
		ASN:         asn,
		State:       "Idle",
		Established: false,
	}
	m.peers[address] = peer

	// Configure peer in GoBGP
	peerConfig := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: address,
			PeerAsn:         asn,
		},
		Transport: &api.Transport{
			RemotePort: uint32(port),
		},
	}

	if err := m.server.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: peerConfig,
	}); err != nil {
		return fmt.Errorf("failed to add peer %s: %w", address, err)
	}

	return nil
}

func (m *Monitor) GetPeer(address string) (*PeerState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[address]
	if !ok {
		return nil, fmt.Errorf("peer %s not found", address)
	}

	peer.mu.RLock()
	defer peer.mu.RUnlock()

	// Create a copy to avoid race conditions
	return &PeerState{
		Address:      peer.Address,
		ASN:          peer.ASN,
		State:        peer.State,
		PrefixCount:  peer.PrefixCount,
		FlapCount:    peer.FlapCount,
		LastFlapTime: peer.LastFlapTime,
		Established:  peer.Established,
	}, nil
}

func (m *Monitor) GetAllPeers() []*PeerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*PeerState, 0, len(m.peers))
	for _, peer := range m.peers {
		peer.mu.RLock()
		peers = append(peers, &PeerState{
			Address:      peer.Address,
			ASN:          peer.ASN,
			State:        peer.State,
			PrefixCount:  peer.PrefixCount,
			FlapCount:    peer.FlapCount,
			LastFlapTime: peer.LastFlapTime,
			Established:  peer.Established,
		})
		peer.mu.RUnlock()
	}
	return peers
}

func (m *Monitor) monitorPeers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updatePeerStates()
		}
	}
}

func (m *Monitor) updatePeerStates() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for address, peer := range m.peers {
		// Get peer state from GoBGP
		req := &api.GetPeerRequest{
			Address: address,
		}

		resp, err := m.server.GetPeer(context.Background(), req)
		if err != nil {
			continue
		}

		peer.mu.Lock()
		wasEstablished := peer.Established

		if resp.Peer.State.SessionState == api.PeerState_ESTABLISHED {
			peer.State = "Established"
			peer.Established = true
			// Count prefixes
			peer.PrefixCount = int64(resp.Peer.State.AdjTable.Advertised)
		} else {
			peer.State = resp.Peer.State.SessionState.String()
			peer.Established = false
		}

		// Detect flaps
		if wasEstablished && !peer.Established {
			peer.FlapCount++
			peer.LastFlapTime = time.Now()
		}

		peer.mu.Unlock()
	}
}

func (m *Monitor) WithdrawAllPrefixes(address string) error {
	m.mu.RLock()
	peer, ok := m.peers[address]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("peer %s not found", address)
	}

	// Get all paths from the peer
	req := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family: &api.Family{
			Afi:  api.Family_AFI_IP,
			Safi: api.Family_SAFI_UNICAST,
		},
	}

	stream, err := m.server.ListPath(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to list paths: %w", err)
	}

	for {
		path, err := stream.Recv()
		if err != nil {
			break
		}

		// Withdraw path
		withdraw := &api.Path{
			Family: path.Path.Family,
			Nlri:   path.Path.Nlri,
		}

		if err := m.server.DeletePath(context.Background(), &api.DeletePathRequest{
			Path: withdraw,
		}); err != nil {
			continue
		}
	}

	peer.mu.Lock()
	peer.FlapCount = 0
	peer.mu.Unlock()

	return nil
}

func (m *Monitor) Close() {
	m.cancel()
	m.server.Stop()
}

// GetPeerMetrics returns metrics for Prometheus
func (m *Monitor) GetPeerMetrics() map[string]float64 {
	metrics := make(map[string]float64)
	peers := m.GetAllPeers()

	for _, peer := range peers {
		key := fmt.Sprintf("bgp_peer_up{peer=\"%s\"}", peer.Address)
		if peer.Established {
			metrics[key] = 1
		} else {
			metrics[key] = 0
		}

		key = fmt.Sprintf("bgp_prefix_count{peer=\"%s\",afi=\"ipv4\"}", peer.Address)
		metrics[key] = float64(peer.PrefixCount)

		key = fmt.Sprintf("bgp_session_flaps_total{peer=\"%s\"}", peer.Address)
		metrics[key] = float64(peer.FlapCount)
	}

	return metrics
}

