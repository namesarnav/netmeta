# netmeta

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Network Intelligence and Auto-Remediation Platform**

`netmeta` is a production-grade Go application that monitors and manages **BGP, OSPF, and MPLS** in multi-vendor network environments. It provides real-time monitoring, topology visualization, and automated remediation capabilities.

## Features

- 🔍 **BGP Monitoring**: Real-time peer state tracking, prefix counting, and flap detection using GoBGP
- 🌐 **OSPF Topology**: Live packet parsing and in-memory topology graph construction
- 🏷️ **MPLS Validation**: Label stack validation and corruption detection
- 🤖 **Auto-Remediation**: Rule-based engine for automatic network issue resolution
- 📊 **Prometheus Metrics**: Comprehensive metrics export for monitoring
- 🖥️ **Web Dashboard**: Real-time WebSocket-based UI with topology visualization
- 🐳 **Container Ready**: Docker and Kubernetes deployment configurations
- 📈 **Grafana Integration**: Pre-configured dashboards for network metrics

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    netmeta Platform                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │   BGP    │  │   OSPF   │  │   MPLS   │             │
│  │ Monitor  │  │  Parser  │  │Validator │             │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
│       │             │             │                     │
│       └─────────────┼─────────────┘                     │
│                     │                                   │
│              ┌──────▼──────┐                           │
│              │ Auto-Remed  │                           │
│              │   Engine    │                           │
│              └──────┬──────┘                           │
│                     │                                   │
│       ┌─────────────┼─────────────┐                     │
│       │             │             │                     │
│  ┌────▼────┐  ┌────▼────┐  ┌────▼────┐               │
│  │Prometheus│  │   Web   │  │   API   │               │
│  │ Exporter │  │   UI    │  │  Layer  │               │
│  └─────────┘  └─────────┘  └─────────┘               │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Installation

### Prerequisites

- Go 1.21 or later
- Docker (optional, for containerized deployment)
- Kubernetes cluster (optional, for K8s deployment)

### Build from Source

```bash
git clone https://github.com/namesarnav/netmeta.git
cd netmeta
go mod download
go build -o netmeta ./cmd/netmeta
```

### Docker

```bash
docker build -t netmeta:latest -f deploy/docker/Dockerfile .
docker run -p 8080:8080 netmeta:latest serve
```

### Kubernetes

```bash
kubectl apply -f deploy/k8s/deployment.yaml
```

## Configuration

Create a `config.yaml` file (see `config.yaml` for example):

```yaml
bgp:
  peers:
    - address: 10.0.0.1
      asn: 65001
      port: 179

ospf:
  interface: eth0
  pcap_file: capture.pcap

mpls:
  enabled: true

auto:
  enabled: true
  flap_threshold: 3
  flap_window_sec: 300

api:
  host: 0.0.0.0
  port: 8080

db:
  path: /var/lib/netmeta
```

## Usage

### CLI Commands

```bash
# Start the server (API + UI)
netmeta serve

# List BGP peers
netmeta bgp peers

# Show OSPF topology
netmeta ospf topology

# Trigger manual remediation
netmeta remediate --peer 10.0.0.1 --reason flap
netmeta remediate --prefix 203.0.113.0/24 --reason rpki

# Show version
netmeta version
```

### Web Dashboard

Access the dashboard at: `http://localhost:8080/dashboard`

Features:
- Real-time BGP peer status
- Live OSPF topology graph
- Remediation event stream
- WebSocket-based updates

### API Endpoints

- `GET /api/v1/bgp/peers` - List all BGP peers
- `GET /api/v1/ospf/topology` - Get OSPF topology
- `GET /api/v1/remediation/events` - Get remediation events
- `GET /metrics` - Prometheus metrics
- `GET /ws` - WebSocket stream

### Prometheus Metrics

Key metrics exported:

- `bgp_peer_up{peer="..."}` - BGP peer up status (1=up, 0=down)
- `bgp_prefix_count{peer="...", afi="ipv4"}` - Prefix count per peer
- `bgp_session_flaps_total{peer="..."}` - Total session flaps
- `mpls_corruption_events_total` - MPLS corruption events
- `netmeta_remediation_total{reason="...", success="..."}` - Remediation actions

## Auto-Remediation

The auto-remediation engine monitors network conditions and automatically triggers remediation actions:

- **BGP Flaps**: If a peer flaps more than 3 times in 5 minutes, all prefixes are withdrawn
- **RPKI Invalid**: Invalid prefixes are automatically withdrawn
- **OSPF Adjacency**: Down adjacencies trigger interface restarts

Rules can be configured in `config.yaml`:

```yaml
auto:
  enabled: true
  flap_threshold: 3
  flap_window_sec: 300
```

## Demo

Run the demo script to see netmeta in action:

```bash
./scripts/demo.sh
```

This will:
1. Start the netmeta server
2. List BGP peers
3. Show OSPF topology
4. Trigger a remediation action
5. Display access URLs

## Development

### Project Structure

```
netmeta/
├── cmd/netmeta/          # Main application entry point
├── pkg/
│   ├── api/              # REST + gRPC handlers
│   ├── auto/             # Auto-remediation logic
│   ├── bgp/              # GoBGP wrapper + monitoring
│   ├── mpls/             # MPLS label validation
│   ├── ospf/             # OSPF packet parser + topology
│   ├── monitor/          # Prometheus exporter
│   └── ui/               # Gin + WebSocket dashboard
├── internal/
│   ├── config/           # Viper config
│   ├── db/               # BadgerDB for state
│   └── telemetry/        # Event logging
├── deploy/
│   ├── docker/           # Dockerfile
│   ├── k8s/              # Kubernetes manifests
│   └── grafana/          # Grafana dashboard
└── scripts/              # Utility scripts
```

### Testing with Real Peers

Use **Containerlab** or **FRR** in Docker to set up test BGP/OSPF peers:

```bash
# Example with FRR
docker run -d --name frr-router frrouting/frr
```

## Deployment

### Docker

```bash
docker build -t netmeta:latest -f deploy/docker/Dockerfile .
docker run -d -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/netmeta/config.yaml \
  netmeta:latest serve
```

### Kubernetes

```bash
kubectl apply -f deploy/k8s/deployment.yaml
```

The deployment includes:
- 3 replicas for high availability
- Prometheus scraping annotations
- Health checks (liveness/readiness probes)
- ConfigMap for configuration

### Grafana Dashboard

Import the dashboard from `deploy/grafana/dashboard.json` into your Grafana instance.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Author

Built for Meta NPE (Network Production Engineering) interview demonstration.

---

**Note**: This is a demonstration project showcasing production-grade network tooling capabilities in Go.

