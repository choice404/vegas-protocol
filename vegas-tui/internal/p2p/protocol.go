package p2p

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Message types exchanged between peers.
const (
	MsgAuthChallenge = "auth_challenge"
	MsgAuthResponse  = "auth_response"
	MsgAuthOK        = "auth_ok"
	MsgAuthFail      = "auth_fail"
	MsgPeerJoined    = "peer_joined"
	MsgPeerLeft      = "peer_left"
	MsgChat          = "chat"
	MsgGameInvite    = "game_invite"
	MsgGameState     = "game_state"
	MsgGameAction    = "game_action"
	MsgGameEnd       = "game_end"
)

// Envelope is the wire format for all P2P messages.
type Envelope struct {
	Type      string          `json:"type"`
	From      string          `json:"from"`
	FromID    string          `json:"from_id"`
	Timestamp int64           `json:"ts"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// NewEnvelope creates an envelope with the current timestamp.
func NewEnvelope(msgType, from, fromID string, payload interface{}) (Envelope, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Envelope{}, err
		}
		raw = b
	}
	return Envelope{
		Type:      msgType,
		From:      from,
		FromID:    fromID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   raw,
	}, nil
}

// Marshal serializes an envelope to a JSON line (with trailing newline).
func Marshal(e Envelope) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Unmarshal deserializes a JSON line into an envelope.
func Unmarshal(line []byte) (Envelope, error) {
	var e Envelope
	err := json.Unmarshal(line, &e)
	return e, err
}

// DecodePayload unmarshals the payload of an envelope into the given type.
func DecodePayload[T any](e Envelope) (T, error) {
	var v T
	if e.Payload == nil {
		return v, fmt.Errorf("nil payload")
	}
	err := json.Unmarshal(e.Payload, &v)
	return v, err
}

// --- Payload types ---

// AuthChallenge is sent by the host to a joining peer.
type AuthChallenge struct {
	Salt      string `json:"salt"`
	Challenge string `json:"challenge"`
}

// AuthResponse is sent by the joiner back to the host.
type AuthResponse struct {
	Name string `json:"name"`
	HMAC string `json:"hmac"`
}

// AuthOK is sent by the host when authentication succeeds.
type AuthOK struct {
	PeerID string   `json:"peer_id"`
	Peers  []string `json:"peers"`
}

// AuthFail is sent by the host when authentication fails.
type AuthFail struct {
	Reason string `json:"reason"`
}

// PeerNotice is sent when a peer joins or leaves.
type PeerNotice struct {
	PeerID string `json:"peer_id"`
	Name   string `json:"name"`
}

// ChatPayload carries a chat message.
type ChatPayload struct {
	Text string `json:"text"`
}

// GameInvite proposes starting a game.
type GameInvite struct {
	Game string `json:"game"`
}

// GameAction carries a player action in a game.
type GameAction struct {
	Action string `json:"action"`
	Amount int    `json:"amount,omitempty"`
}

// --- Crypto helpers ---

// DeriveKey derives a symmetric key from a passphrase and salt using SHA-256.
func DeriveKey(passphrase, salt string) []byte {
	h := sha256.New()
	h.Write([]byte(passphrase))
	h.Write([]byte(salt))
	return h.Sum(nil)
}

// ComputeHMAC computes HMAC-SHA256 of the challenge using the derived key.
func ComputeHMAC(challenge string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC checks that the provided HMAC matches the expected value.
func VerifyHMAC(challenge, provided string, key []byte) bool {
	expected := ComputeHMAC(challenge, key)
	return hmac.Equal([]byte(expected), []byte(provided))
}

// RandomHex generates a random hex string of n bytes.
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
