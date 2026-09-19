package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"pushly/internal/protocol"
)

type RelayServer struct {
	Pairing         *PairingManager
	Replay          *protocol.ReplayCache
	mu              sync.RWMutex
	agentChannels   map[string]chan *protocol.Message // agentID -> channel
	responseWaiters map[string]chan *protocol.Message // messageID -> channel
}

func NewRelayServer() *RelayServer {
	return &RelayServer{
		Pairing:         NewPairingManager(),
		Replay:          protocol.NewReplayCache(5 * time.Minute),
		agentChannels:   make(map[string]chan *protocol.Message),
		responseWaiters: make(map[string]chan *protocol.Message),
	}
}

func (s *RelayServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/agent/pair/code", s.handleAgentGeneratePairCode)
	mux.HandleFunc("/api/v1/agent/events", s.handleAgentEvents)
	mux.HandleFunc("/api/v1/agent/response", s.handleAgentResponse)
	mux.HandleFunc("/api/v1/client/pair", s.handleClientPair)
	mux.HandleFunc("/api/v1/client/command", s.handleClientCommand)
	return mux
}

func (s *RelayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

func (s *RelayServer) handleAgentGeneratePairCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string `json:"agent_id"`
		AgentName string `json:"agent_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	code, err := s.Pairing.GeneratePairingCode(req.AgentID, req.AgentName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pairing_code": code,
		"expires_in":   300,
	})
}

func (s *RelayServer) handleClientPair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req protocol.PairRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	paired, err := s.Pairing.CompletePairing(req.PairingCode, req.DeviceID, req.DeviceName)
	if err != nil {
		http.Error(w, "pairing failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	resp := protocol.PairResponsePayload{
		AgentID:      paired.AgentID,
		AgentName:    paired.DeviceName,
		SessionID:    paired.SessionID,
		SessionToken: paired.SessionToken,
		SharedKeyHex: paired.SharedKeyHex,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *RelayServer) handleAgentEvents(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id query parameter is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	msgChan := make(chan *protocol.Message, 16)
	s.mu.Lock()
	s.agentChannels[agentID] = msgChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.agentChannels[agentID] == msgChan {
			delete(s.agentChannels, agentID)
		}
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Heartbeat
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()
		case msg := <-msgChan:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}

func (s *RelayServer) handleAgentResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	waiter, exists := s.responseWaiters[msg.ID]
	s.mu.RUnlock()

	if exists {
		select {
		case waiter <- &msg:
		default:
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"received": true})
}

func (s *RelayServer) handleClientCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := msg.Validate(); err != nil {
		http.Error(w, "validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate session
	token := r.Header.Get("X-Session-Token")
	paired, err := s.Pairing.ValidateSession(msg.SessionID, token)
	if err != nil {
		http.Error(w, "session unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Replay protection
	if !s.Replay.CheckAndRecord(msg.ID) {
		http.Error(w, "replay detected: duplicate message id", http.StatusConflict)
		return
	}

	msg.AgentID = paired.AgentID
	msg.DeviceID = paired.DeviceID

	s.mu.RLock()
	agentChan, agentOnline := s.agentChannels[paired.AgentID]
	s.mu.RUnlock()

	if !agentOnline {
		http.Error(w, "agent is offline", http.StatusServiceUnavailable)
		return
	}

	responseChan := make(chan *protocol.Message, 1)
	s.mu.Lock()
	s.responseWaiters[msg.ID] = responseChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.responseWaiters, msg.ID)
		s.mu.Unlock()
	}()

	// Send message to agent
	select {
	case agentChan <- &msg:
	default:
		http.Error(w, "agent channel busy", http.StatusServiceUnavailable)
		return
	}

	// Wait for agent response
	select {
	case <-r.Context().Done():
		return
	case <-time.After(45 * time.Second):
		http.Error(w, "agent command timeout", http.StatusGatewayTimeout)
		return
	case resp := <-responseChan:
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
