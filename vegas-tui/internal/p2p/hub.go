package p2p

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"
)

// Peer represents a connected peer.
type Peer struct {
	ID   string
	Name string
	conn net.Conn
}

// Hub manages P2P connections. It can act as host (server) or joiner (client).
type Hub struct {
	mu         sync.RWMutex
	peers      map[string]*Peer
	localName  string
	localID    string
	passphrase string
	isHost     bool
	listener   net.Listener
	stopped    bool
	IncomingCh chan Envelope
}

// NewHub creates a new P2P hub.
func NewHub() *Hub {
	id, _ := RandomHex(8)
	return &Hub{
		peers:      make(map[string]*Peer),
		localID:    id,
		IncomingCh: make(chan Envelope, 64),
	}
}

// LocalID returns this hub's peer ID.
func (h *Hub) LocalID() string {
	return h.localID
}

// IsHost returns whether this hub is hosting.
func (h *Hub) IsHost() bool {
	return h.isHost
}

// Peers returns a snapshot of connected peers.
func (h *Hub) Peers() []Peer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]Peer, 0, len(h.peers))
	for _, p := range h.peers {
		out = append(out, Peer{ID: p.ID, Name: p.Name})
	}
	return out
}

// Host starts a TLS server on the given port with the given passphrase.
func (h *Hub) Host(name string, port int, passphrase string) (string, error) {
	h.mu.Lock()
	h.localName = name
	h.passphrase = passphrase
	h.isHost = true
	h.stopped = false
	h.mu.Unlock()

	tlsCfg, err := generateTLSConfig()
	if err != nil {
		return "", fmt.Errorf("TLS setup: %w", err)
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsCfg)
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}
	h.mu.Lock()
	h.listener = ln
	h.mu.Unlock()

	ip := LocalIP()
	addr := fmt.Sprintf("%s:%d", ip, port)

	go h.acceptLoop(ln)

	return addr, nil
}

// Join connects to a host at the given address with the given passphrase.
func (h *Hub) Join(name, address, passphrase string) error {
	h.mu.Lock()
	h.localName = name
	h.passphrase = passphrase
	h.isHost = false
	h.stopped = false
	h.mu.Unlock()

	tlsCfg := &tls.Config{InsecureSkipVerify: true} // self-signed certs
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, tlsCfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Complete auth handshake as joiner
	if err := h.joinerHandshake(conn, name, passphrase); err != nil {
		conn.Close()
		return err
	}

	return nil
}

// Send broadcasts a message to all connected peers.
func (h *Hub) Send(e Envelope) error {
	data, err := Marshal(e)
	if err != nil {
		return err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		p.conn.Write(data)
	}
	return nil
}

// SendTo sends a message to a specific peer.
func (h *Hub) SendTo(peerID string, e Envelope) error {
	data, err := Marshal(e)
	if err != nil {
		return err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if p, ok := h.peers[peerID]; ok {
		_, err = p.conn.Write(data)
		return err
	}
	return fmt.Errorf("peer %s not found", peerID)
}

// Stop shuts down all connections.
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopped = true
	if h.listener != nil {
		h.listener.Close()
		h.listener = nil
	}
	for id, p := range h.peers {
		p.conn.Close()
		delete(h.peers, id)
	}
}

func (h *Hub) isStopped() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.stopped
}

func (h *Hub) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if h.isStopped() {
				return
			}
			continue
		}
		go h.hostHandshake(conn)
	}
}

func (h *Hub) hostHandshake(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	salt, _ := RandomHex(16)
	challenge, _ := RandomHex(16)

	// Send challenge
	env, _ := NewEnvelope(MsgAuthChallenge, h.localName, h.localID, AuthChallenge{
		Salt:      salt,
		Challenge: challenge,
	})
	data, _ := Marshal(env)
	conn.Write(data)

	// Read response
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	if !scanner.Scan() {
		conn.Close()
		return
	}
	resp, err := Unmarshal(scanner.Bytes())
	if err != nil || resp.Type != MsgAuthResponse {
		conn.Close()
		return
	}

	authResp, err := DecodePayload[AuthResponse](resp)
	if err != nil {
		conn.Close()
		return
	}

	// Verify HMAC
	key := DeriveKey(h.passphrase, salt)
	if !VerifyHMAC(challenge, authResp.HMAC, key) {
		failEnv, _ := NewEnvelope(MsgAuthFail, h.localName, h.localID, AuthFail{
			Reason: "Invalid passphrase",
		})
		failData, _ := Marshal(failEnv)
		conn.Write(failData)
		conn.Close()
		return
	}

	// Auth success
	peerID := resp.FromID
	peerName := authResp.Name

	// Gather existing peer names
	h.mu.RLock()
	var peerNames []string
	for _, p := range h.peers {
		peerNames = append(peerNames, p.Name)
	}
	peerNames = append(peerNames, h.localName)
	h.mu.RUnlock()

	okEnv, _ := NewEnvelope(MsgAuthOK, h.localName, h.localID, AuthOK{
		PeerID: peerID,
		Peers:  peerNames,
	})
	okData, _ := Marshal(okEnv)
	conn.Write(okData)

	conn.SetDeadline(time.Time{}) // Clear deadline

	// Register peer
	peer := &Peer{ID: peerID, Name: peerName, conn: conn}
	h.mu.Lock()
	h.peers[peerID] = peer
	h.mu.Unlock()

	// Notify existing peers
	joinEnv, _ := NewEnvelope(MsgPeerJoined, peerName, peerID, PeerNotice{
		PeerID: peerID,
		Name:   peerName,
	})
	h.broadcastExcept(peerID, joinEnv)

	// Deliver join notification locally
	h.deliver(joinEnv)

	// Read loop
	h.readLoop(peer, scanner)
}

func (h *Hub) joinerHandshake(conn net.Conn, name, passphrase string) error {
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	// Read challenge from host
	if !scanner.Scan() {
		return fmt.Errorf("no challenge received")
	}
	challengeEnv, err := Unmarshal(scanner.Bytes())
	if err != nil || challengeEnv.Type != MsgAuthChallenge {
		return fmt.Errorf("unexpected message during handshake")
	}

	challenge, err := DecodePayload[AuthChallenge](challengeEnv)
	if err != nil {
		return fmt.Errorf("bad challenge: %w", err)
	}

	// Compute HMAC and send response
	key := DeriveKey(passphrase, challenge.Salt)
	mac := ComputeHMAC(challenge.Challenge, key)

	respEnv, _ := NewEnvelope(MsgAuthResponse, name, h.localID, AuthResponse{
		Name: name,
		HMAC: mac,
	})
	respData, _ := Marshal(respEnv)
	conn.Write(respData)

	// Read auth result
	if !scanner.Scan() {
		return fmt.Errorf("no auth result received")
	}
	resultEnv, err := Unmarshal(scanner.Bytes())
	if err != nil {
		return fmt.Errorf("bad auth result: %w", err)
	}

	if resultEnv.Type == MsgAuthFail {
		fail, _ := DecodePayload[AuthFail](resultEnv)
		return fmt.Errorf("authentication failed: %s", fail.Reason)
	}
	if resultEnv.Type != MsgAuthOK {
		return fmt.Errorf("unexpected auth result: %s", resultEnv.Type)
	}

	conn.SetDeadline(time.Time{})

	// Register host as a peer
	hostPeer := &Peer{
		ID:   resultEnv.FromID,
		Name: resultEnv.From,
		conn: conn,
	}
	h.mu.Lock()
	h.peers[hostPeer.ID] = hostPeer
	h.mu.Unlock()

	// Deliver the auth_ok so the TUI knows we're connected
	h.deliver(resultEnv)

	go h.readLoop(hostPeer, scanner)

	return nil
}

func (h *Hub) readLoop(peer *Peer, scanner *bufio.Scanner) {
	for scanner.Scan() {
		if h.isStopped() {
			return
		}
		env, err := Unmarshal(scanner.Bytes())
		if err != nil {
			continue
		}

		// Host relays messages to other peers
		if h.isHost {
			h.broadcastExcept(peer.ID, env)
		}

		h.deliver(env)
	}

	// Peer disconnected
	h.mu.Lock()
	delete(h.peers, peer.ID)
	h.mu.Unlock()

	if h.isStopped() {
		return
	}

	leftEnv, _ := NewEnvelope(MsgPeerLeft, peer.Name, peer.ID, PeerNotice{
		PeerID: peer.ID,
		Name:   peer.Name,
	})
	if h.isHost {
		h.broadcastExcept(peer.ID, leftEnv)
	}
	h.deliver(leftEnv)
}

func (h *Hub) deliver(e Envelope) {
	select {
	case h.IncomingCh <- e:
	default:
		// Drop if channel full
	}
}

func (h *Hub) broadcastExcept(excludeID string, e Envelope) {
	data, err := Marshal(e)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, p := range h.peers {
		if id != excludeID {
			p.conn.Write(data)
		}
	}
}

// LocalIP returns the first non-loopback IPv4 address found.
func LocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
