#!/bin/bash

set -e

echo "=========================================="
echo "netmeta Demo Script"
echo "=========================================="
echo ""

# Check if netmeta binary exists
if [ ! -f "./netmeta" ]; then
    echo "Building netmeta..."
    go build -o netmeta ./cmd/netmeta
fi

echo "Starting netmeta server..."
./netmeta serve &
NETMETA_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 5

# Check if server is running
if ! kill -0 $NETMETA_PID 2>/dev/null; then
    echo "Error: netmeta server failed to start"
    exit 1
fi

echo "Server started (PID: $NETMETA_PID)"
echo ""

# List BGP peers
echo "Listing BGP peers..."
./netmeta bgp peers
echo ""

# Show OSPF topology
echo "Showing OSPF topology..."
./netmeta ospf topology
echo ""

# Simulate remediation
echo "Triggering manual remediation..."
./netmeta remediate --peer 10.0.0.1 --reason test
echo ""

echo "=========================================="
echo "Demo completed!"
echo ""
echo "Access the dashboard at: http://localhost:8080/dashboard"
echo "Metrics endpoint: http://localhost:8080/metrics"
echo "API endpoint: http://localhost:8080/api/v1/bgp/peers"
echo ""
echo "Check Grafana dashboard at: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop the server"
echo "=========================================="

# Wait for user interrupt
trap "kill $NETMETA_PID; exit" INT TERM
wait $NETMETA_PID

