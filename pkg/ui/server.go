package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/namesarnav/netmeta/internal/config"
	"github.com/namesarnav/netmeta/pkg/auto"
	"github.com/namesarnav/netmeta/pkg/bgp"
	"github.com/namesarnav/netmeta/pkg/ospf"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	cfg        *config.Config
	bgpMonitor *bgp.Monitor
	ospfParser *ospf.Parser
	autoEngine *auto.Engine
	router     *gin.Engine
}

func NewServer(cfg *config.Config, bgpMonitor *bgp.Monitor, ospfParser *ospf.Parser, autoEngine *auto.Engine) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	s := &Server{
		cfg:        cfg,
		bgpMonitor: bgpMonitor,
		ospfParser: ospfParser,
		autoEngine: autoEngine,
		router:     router,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Static files
	s.router.Static("/static", "./static")
	s.router.LoadHTMLGlob("templates/*")

	// Dashboard
	s.router.GET("/dashboard", s.handleDashboard)

	// WebSocket
	s.router.GET("/ws", s.handleWebSocket)

	// API endpoints
	api := s.router.Group("/api/v1")
	{
		api.GET("/bgp/peers", s.handleBGPPeers)
		api.GET("/ospf/topology", s.handleOSPFTopology)
		api.GET("/remediation/events", s.handleRemediationEvents)
	}
}

func (s *Server) handleDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "netmeta Dashboard",
	})
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send BGP peer updates
			peers := s.bgpMonitor.GetAllPeers()
			peerData := make([]map[string]interface{}, len(peers))
			for i, peer := range peers {
				peerData[i] = map[string]interface{}{
					"address":     peer.Address,
					"asn":         peer.ASN,
					"state":       peer.State,
					"prefixCount": peer.PrefixCount,
					"flapCount":   peer.FlapCount,
					"established": peer.Established,
				}
			}

			// Send OSPF topology
			topology := s.ospfParser.GetTopology()
			topoData := make(map[string]interface{})
			for routerID, links := range topology.Routers {
				linkData := make([]map[string]interface{}, len(links))
				for i, link := range links {
					linkData[i] = map[string]interface{}{
						"remoteRouterID": link.RemoteRouterID,
						"cost":           link.Cost,
						"state":          link.State,
					}
				}
				topoData[fmt.Sprintf("%d", routerID)] = linkData
			}

			// Send remediation events
			events := s.autoEngine.GetEvents(10)
			eventData := make([]map[string]interface{}, len(events))
			for i, event := range events {
				eventData[i] = map[string]interface{}{
					"timestamp": event.Timestamp,
					"type":      event.Type,
					"target":    event.Target,
					"reason":    event.Reason,
					"action":    event.Action,
					"success":   event.Success,
				}
			}

			message := map[string]interface{}{
				"type":      "update",
				"peers":     peerData,
				"topology":  topoData,
				"events":    eventData,
				"timestamp": time.Now(),
			}

			if err := conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

func (s *Server) handleBGPPeers(c *gin.Context) {
	peers := s.bgpMonitor.GetAllPeers()
	c.JSON(http.StatusOK, peers)
}

func (s *Server) handleOSPFTopology(c *gin.Context) {
	topology := s.ospfParser.GetTopology()
	c.JSON(http.StatusOK, topology)
}

func (s *Server) handleRemediationEvents(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := json.Marshal(l); err == nil {
			_ = parsed // Would parse limit here
		}
	}
	events := s.autoEngine.GetEvents(limit)
	c.JSON(http.StatusOK, events)
}

func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.API.Host, s.cfg.API.Port)
	log.Printf("Starting UI server on %s", addr)
	return s.router.Run(addr)
}

