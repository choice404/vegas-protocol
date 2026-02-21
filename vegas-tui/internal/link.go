package internal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/games"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/p2p"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// --- BubbleTea messages ---

type p2pIncomingMsg struct{ Envelope p2p.Envelope }
type p2pErrorMsg struct{ Err error }
type p2pHostedMsg struct{ Address string }
type p2pJoinedMsg struct{}

// --- Link UI states ---

type linkState int

const (
	linkLobby linkState = iota
	linkHosting
	linkJoining
	linkConnected
	linkChat
	linkPoker
)

// lobbyField identifies the form fields in lobby/joining views.
type lobbyField struct {
	Key      string
	Label    string
	Value    string
	IsPassword bool
}

// LinkModel implements the LINK tab.
type LinkModel struct {
	state       linkState
	hub         *p2p.Hub
	appSettings *settings.Settings

	// Lobby form (settings-style cursor + enter-to-edit)
	lobbyFields []lobbyField
	lobbyCursor int
	lobbyEditing bool
	input       textinput.Model

	// Stored values (persisted across state changes)
	nameValue string
	passValue string
	addrValue string

	// Chat
	chatInput    textinput.Model
	chatMessages []chatMsg
	chatViewport viewport.Model

	// Connection info
	hostAddress string
	peerNames   []string
	statusMsg   string
	errorMsg    string

	// Poker
	pokerGame  *games.HoldemGame // non-nil only on host
	pokerState *games.HoldemState
	raiseInput textinput.Model

	// Layout
	width  int
	height int
}

type chatMsg struct {
	Sender string
	Text   string
	Time   time.Time
}

func NewLinkModel(s *settings.Settings) LinkModel {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 40

	chatIn := textinput.New()
	chatIn.Placeholder = "Type message..."
	chatIn.CharLimit = 300
	chatIn.Width = 50

	raiseIn := textinput.New()
	raiseIn.Placeholder = "Amount"
	raiseIn.CharLimit = 10
	raiseIn.Width = 10

	m := LinkModel{
		state:       linkLobby,
		appSettings: s,
		input:       ti,
		chatInput:   chatIn,
		raiseInput:  raiseIn,
		nameValue:   s.DisplayName,
	}
	m.refreshLobbyFields()
	return m
}

func (m *LinkModel) refreshLobbyFields() {
	if m.state == linkJoining {
		m.lobbyFields = []lobbyField{
			{Key: "name", Label: "DISPLAY NAME", Value: m.nameValue},
			{Key: "pass", Label: "PASSPHRASE", Value: m.passValue, IsPassword: true},
			{Key: "addr", Label: "HOST ADDRESS", Value: m.addrValue},
		}
	} else {
		m.lobbyFields = []lobbyField{
			{Key: "name", Label: "DISPLAY NAME", Value: m.nameValue},
			{Key: "pass", Label: "PASSPHRASE", Value: m.passValue, IsPassword: true},
		}
	}
}

func (m *LinkModel) applyLobbyField(idx int, val string) {
	if idx >= len(m.lobbyFields) {
		return
	}
	switch m.lobbyFields[idx].Key {
	case "name":
		m.nameValue = val
	case "pass":
		m.passValue = val
	case "addr":
		m.addrValue = val
	}
	m.refreshLobbyFields()
}

func (m LinkModel) Init() tea.Cmd {
	return nil
}

// InputFocused returns true when a text input has focus.
func (m LinkModel) InputFocused() bool {
	switch m.state {
	case linkLobby, linkJoining:
		return m.lobbyEditing
	case linkChat:
		return m.chatInput.Focused()
	case linkPoker:
		return m.raiseInput.Focused()
	}
	return false
}

func (m LinkModel) Update(msg tea.Msg) (LinkModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case p2pIncomingMsg:
		cmd := m.handleP2PMessage(msg.Envelope)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Continue listening
		if m.hub != nil {
			cmds = append(cmds, waitForP2PMsg(m.hub))
		}
		return m, tea.Batch(cmds...)

	case p2pErrorMsg:
		m.errorMsg = msg.Err.Error()
		return m, nil

	case p2pHostedMsg:
		m.hostAddress = msg.Address
		m.state = linkHosting
		cmds = append(cmds, waitForP2PMsg(m.hub))
		return m, tea.Batch(cmds...)

	case p2pJoinedMsg:
		// Join succeeded — start listening for messages (auth_ok is already queued)
		if m.hub != nil {
			cmds = append(cmds, waitForP2PMsg(m.hub))
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			cmd := m.handleClick(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
		// Scroll in lobby
		if m.state == linkLobby || m.state == linkJoining {
			if msg.Button == tea.MouseButtonWheelUp && m.lobbyCursor > 0 {
				m.lobbyCursor--
			}
			if msg.Button == tea.MouseButtonWheelDown && m.lobbyCursor < len(m.lobbyFields)-1 {
				m.lobbyCursor++
			}
		}

	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// Update chat/poker inputs for non-key messages (blink, etc.)
	switch m.state {
	case linkLobby, linkJoining:
		if m.lobbyEditing {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	case linkChat:
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		cmds = append(cmds, cmd)
		m.chatViewport, cmd = m.chatViewport.Update(msg)
		cmds = append(cmds, cmd)
	case linkPoker:
		var cmd tea.Cmd
		m.raiseInput, cmd = m.raiseInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *LinkModel) handleP2PMessage(env p2p.Envelope) tea.Cmd {
	switch env.Type {
	case p2p.MsgAuthOK:
		authOK, err := p2p.DecodePayload[p2p.AuthOK](env)
		if err == nil {
			m.peerNames = authOK.Peers
		}
		m.state = linkConnected
		m.statusMsg = "Connected to " + env.From
		m.addSystemChat("Connected to network. Welcome, Courier.")

	case p2p.MsgPeerJoined:
		notice, _ := p2p.DecodePayload[p2p.PeerNotice](env)
		m.peerNames = append(m.peerNames, notice.Name)
		m.addSystemChat(notice.Name + " has joined the network.")

	case p2p.MsgPeerLeft:
		notice, _ := p2p.DecodePayload[p2p.PeerNotice](env)
		m.removePeerName(notice.Name)
		m.addSystemChat(notice.Name + " has left the network.")
		if m.pokerGame != nil {
			m.pokerGame.RemovePlayer(notice.PeerID)
		}

	case p2p.MsgChat:
		chat, _ := p2p.DecodePayload[p2p.ChatPayload](env)
		m.chatMessages = append(m.chatMessages, chatMsg{
			Sender: env.From,
			Text:   chat.Text,
			Time:   time.Now(),
		})
		m.updateChatViewport()

	case p2p.MsgGameInvite:
		invite, _ := p2p.DecodePayload[p2p.GameInvite](env)
		if invite.Game == "holdem" {
			m.addSystemChat("Poker game starting! Shuffling up and dealing...")
			m.state = linkPoker
		}

	case p2p.MsgGameAction:
		if m.hub != nil && m.hub.IsHost() && m.pokerGame != nil {
			action, _ := p2p.DecodePayload[p2p.GameAction](env)
			err := m.pokerGame.ProcessAction(env.FromID, games.PlayerAction(action.Action), action.Amount)
			if err == nil {
				m.pokerState = ptrState(m.pokerGame.State())
				hub := m.hub
				game := m.pokerGame
				return func() tea.Msg {
					for _, peer := range hub.Peers() {
						state := game.StateForPlayer(peer.ID)
						stateData, _ := json.Marshal(state)
						e, _ := p2p.NewEnvelope(p2p.MsgGameState, "", hub.LocalID(), json.RawMessage(stateData))
						hub.SendTo(peer.ID, e)
					}
					return nil
				}
			}
		}

	case p2p.MsgGameState:
		var state games.HoldemState
		if err := json.Unmarshal(env.Payload, &state); err == nil {
			m.pokerState = &state
			m.state = linkPoker
		}

	case p2p.MsgGameEnd:
		m.pokerGame = nil
		m.pokerState = nil
		m.state = linkConnected
		m.addSystemChat("Poker game ended. Returning to lobby.")
	}
	return nil
}

func (m *LinkModel) handleClick(msg tea.MouseMsg) tea.Cmd {
	switch m.state {
	case linkLobby:
		if zone.Get("link-host").InBounds(msg) {
			return m.doHost()
		}
		if zone.Get("link-join").InBounds(msg) {
			m.state = linkJoining
			m.lobbyCursor = 2 // focus addr field
			m.lobbyEditing = false
			m.refreshLobbyFields()
			return nil
		}
		for i := range m.lobbyFields {
			if zone.Get(fmt.Sprintf("link-field-%d", i)).InBounds(msg) {
				m.lobbyCursor = i
				return nil
			}
		}

	case linkHosting:
		if zone.Get("link-chat").InBounds(msg) {
			m.state = linkChat
			m.chatInput.Focus()
			return nil
		}
		if zone.Get("link-poker").InBounds(msg) {
			return m.startPoker()
		}
		if zone.Get("link-cancel").InBounds(msg) {
			m.disconnect()
			return nil
		}

	case linkJoining:
		if zone.Get("link-connect").InBounds(msg) {
			return m.doJoin()
		}
		if zone.Get("link-back").InBounds(msg) {
			m.state = linkLobby
			m.lobbyEditing = false
			m.refreshLobbyFields()
			return nil
		}
		for i := range m.lobbyFields {
			if zone.Get(fmt.Sprintf("link-field-%d", i)).InBounds(msg) {
				m.lobbyCursor = i
				return nil
			}
		}

	case linkConnected:
		if zone.Get("link-chat").InBounds(msg) {
			m.state = linkChat
			m.chatInput.Focus()
			return nil
		}
		if zone.Get("link-poker").InBounds(msg) {
			return m.startPoker()
		}
		if zone.Get("link-disconnect").InBounds(msg) {
			m.disconnect()
			return nil
		}

	case linkChat:
		if zone.Get("link-send").InBounds(msg) {
			return m.sendChatMessage()
		}
		if zone.Get("link-back-lobby").InBounds(msg) {
			m.state = linkConnected
			m.chatInput.Blur()
			return nil
		}

	case linkPoker:
		return m.handlePokerClick(msg)
	}
	return nil
}

func (m *LinkModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch m.state {
	case linkLobby, linkJoining:
		return m.handleLobbyKey(msg)

	case linkHosting:
		switch msg.String() {
		case "c":
			if len(m.peerNames) > 0 {
				m.state = linkChat
				m.chatInput.Focus()
				return nil
			}
		case "p":
			if len(m.peerNames) > 0 {
				return m.startPoker()
			}
		case "esc":
			m.disconnect()
			return nil
		}

	case linkConnected:
		switch msg.String() {
		case "c":
			m.state = linkChat
			m.chatInput.Focus()
			return nil
		case "p":
			return m.startPoker()
		case "esc":
			m.disconnect()
			return nil
		}

	case linkChat:
		return m.handleChatKey(msg)

	case linkPoker:
		return m.handlePokerKey(msg)
	}
	return nil
}

func (m *LinkModel) handleLobbyKey(msg tea.KeyMsg) tea.Cmd {
	if m.lobbyEditing {
		switch msg.String() {
		case "enter":
			val := m.input.Value()
			m.applyLobbyField(m.lobbyCursor, val)
			m.lobbyEditing = false
			m.input.Blur()
			return nil
		case "esc":
			m.lobbyEditing = false
			m.input.Blur()
			return nil
		}
		// Forward all other keys to the textinput
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}

	// Not editing — navigate
	switch msg.String() {
	case "up", "k":
		if m.lobbyCursor > 0 {
			m.lobbyCursor--
		}
	case "down", "j":
		if m.lobbyCursor < len(m.lobbyFields)-1 {
			m.lobbyCursor++
		}
	case "enter":
		// Start editing the selected field
		m.lobbyEditing = true
		m.input.SetValue(m.lobbyFields[m.lobbyCursor].Value)
		if m.lobbyFields[m.lobbyCursor].IsPassword {
			m.input.EchoMode = textinput.EchoPassword
			m.input.EchoCharacter = '•'
		} else {
			m.input.EchoMode = textinput.EchoNormal
		}
		m.input.Focus()
		return textinput.Blink
	case "h":
		if m.state == linkLobby {
			return m.doHost()
		}
	case "esc":
		if m.state == linkJoining {
			m.state = linkLobby
			m.lobbyEditing = false
			m.refreshLobbyFields()
			return nil
		}
	case "tab":
		if m.state == linkJoining {
			return m.doJoin()
		} else {
			return m.doHost()
		}
	}
	return nil
}

func (m *LinkModel) handleChatKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return m.sendChatMessage()
	case "esc":
		m.state = linkConnected
		m.chatInput.Blur()
		return nil
	}
	// Forward to chat input
	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return cmd
}

// --- Actions ---

func (m *LinkModel) doHost() tea.Cmd {
	name := strings.TrimSpace(m.nameValue)
	pass := strings.TrimSpace(m.passValue)
	if name == "" {
		m.errorMsg = "Enter a display name"
		return nil
	}
	if pass == "" {
		m.errorMsg = "Enter a passphrase"
		return nil
	}
	m.errorMsg = ""

	m.hub = p2p.NewHub()
	port := m.appSettings.P2PPort
	if port == 0 {
		port = 9999
	}

	hub := m.hub
	return func() tea.Msg {
		addr, err := hub.Host(name, port, pass)
		if err != nil {
			return p2pErrorMsg{Err: err}
		}
		return p2pHostedMsg{Address: addr}
	}
}

func (m *LinkModel) doJoin() tea.Cmd {
	name := strings.TrimSpace(m.nameValue)
	pass := strings.TrimSpace(m.passValue)
	addr := strings.TrimSpace(m.addrValue)
	if name == "" || pass == "" || addr == "" {
		m.errorMsg = "All fields required"
		return nil
	}
	m.errorMsg = ""
	m.statusMsg = "Connecting..."

	m.hub = p2p.NewHub()
	hub := m.hub
	return func() tea.Msg {
		err := hub.Join(name, addr, pass)
		if err != nil {
			return p2pErrorMsg{Err: err}
		}
		return p2pJoinedMsg{}
	}
}

func (m *LinkModel) disconnect() {
	if m.hub != nil {
		m.hub.Stop()
		m.hub = nil
	}
	m.state = linkLobby
	m.peerNames = nil
	m.hostAddress = ""
	m.statusMsg = ""
	m.errorMsg = ""
	m.pokerGame = nil
	m.pokerState = nil
	m.chatMessages = nil
	m.lobbyEditing = false
	m.refreshLobbyFields()
}

func (m *LinkModel) sendChatMessage() tea.Cmd {
	text := strings.TrimSpace(m.chatInput.Value())
	if text == "" || m.hub == nil {
		return nil
	}
	m.chatInput.Reset()

	m.chatMessages = append(m.chatMessages, chatMsg{
		Sender: "You",
		Text:   text,
		Time:   time.Now(),
	})
	m.updateChatViewport()

	hub := m.hub
	return func() tea.Msg {
		env, _ := p2p.NewEnvelope(p2p.MsgChat, "", hub.LocalID(), p2p.ChatPayload{Text: text})
		if err := hub.Send(env); err != nil {
			return p2pErrorMsg{Err: err}
		}
		return nil
	}
}

func (m *LinkModel) startPoker() tea.Cmd {
	if m.hub == nil {
		return nil
	}

	peers := m.hub.Peers()
	if len(peers) < 1 {
		m.errorMsg = "Need at least 2 players"
		return nil
	}

	if m.hub.IsHost() {
		players := []games.Player{
			{ID: m.hub.LocalID(), Name: strings.TrimSpace(m.nameValue), Chips: 1000},
		}
		for _, peer := range peers {
			players = append(players, games.Player{ID: peer.ID, Name: peer.Name, Chips: 1000})
		}
		m.pokerGame = games.NewHoldemGame(players, 10, 20)
		m.pokerGame.StartHand()

		hub := m.hub
		game := m.pokerGame
		m.pokerState = ptrState(game.State())
		m.state = linkPoker

		return func() tea.Msg {
			inviteEnv, _ := p2p.NewEnvelope(p2p.MsgGameInvite, "", hub.LocalID(), p2p.GameInvite{Game: "holdem"})
			hub.Send(inviteEnv)

			for _, peer := range hub.Peers() {
				state := game.StateForPlayer(peer.ID)
				stateData, _ := json.Marshal(state)
				env, _ := p2p.NewEnvelope(p2p.MsgGameState, "", hub.LocalID(), json.RawMessage(stateData))
				hub.SendTo(peer.ID, env)
			}
			return nil
		}
	}

	m.errorMsg = "Only the host can start games"
	return nil
}

func (m *LinkModel) handlePokerClick(msg tea.MouseMsg) tea.Cmd {
	if zone.Get("poker-fold").InBounds(msg) {
		return m.pokerAction(games.ActionFold, 0)
	}
	if zone.Get("poker-check").InBounds(msg) {
		return m.pokerAction(games.ActionCheck, 0)
	}
	if zone.Get("poker-call").InBounds(msg) {
		return m.pokerAction(games.ActionCall, 0)
	}
	if zone.Get("poker-raise").InBounds(msg) {
		amt := 0
		fmt.Sscanf(m.raiseInput.Value(), "%d", &amt)
		if amt > 0 {
			m.raiseInput.Reset()
			return m.pokerAction(games.ActionRaise, amt)
		}
		return nil
	}
	if zone.Get("poker-allin").InBounds(msg) {
		return m.pokerAction(games.ActionAllIn, 0)
	}
	if zone.Get("poker-back").InBounds(msg) {
		m.state = linkConnected
		return nil
	}
	if zone.Get("poker-deal").InBounds(msg) {
		return m.dealNextHand()
	}
	return nil
}

func (m *LinkModel) handlePokerKey(msg tea.KeyMsg) tea.Cmd {
	if m.raiseInput.Focused() {
		switch msg.String() {
		case "enter":
			amt := 0
			fmt.Sscanf(m.raiseInput.Value(), "%d", &amt)
			m.raiseInput.Reset()
			m.raiseInput.Blur()
			if amt > 0 {
				return m.pokerAction(games.ActionRaise, amt)
			}
			return nil
		case "esc":
			m.raiseInput.Blur()
			return nil
		}
		var cmd tea.Cmd
		m.raiseInput, cmd = m.raiseInput.Update(msg)
		return cmd
	}

	switch msg.String() {
	case "f":
		return m.pokerAction(games.ActionFold, 0)
	case "k":
		return m.pokerAction(games.ActionCheck, 0)
	case "c":
		return m.pokerAction(games.ActionCall, 0)
	case "a":
		return m.pokerAction(games.ActionAllIn, 0)
	case "r":
		m.raiseInput.Focus()
		return textinput.Blink
	case "enter":
		if m.pokerState != nil && m.pokerState.Phase == games.PhaseHandOver {
			return m.dealNextHand()
		}
	case "esc":
		m.state = linkConnected
		return nil
	}
	return nil
}

func (m *LinkModel) pokerAction(action games.PlayerAction, amount int) tea.Cmd {
	if m.hub == nil {
		return nil
	}

	if m.hub.IsHost() && m.pokerGame != nil {
		err := m.pokerGame.ProcessAction(m.hub.LocalID(), action, amount)
		if err != nil {
			m.errorMsg = err.Error()
			return nil
		}
		m.errorMsg = ""
		m.pokerState = ptrState(m.pokerGame.State())

		hub := m.hub
		game := m.pokerGame
		return func() tea.Msg {
			for _, peer := range hub.Peers() {
				state := game.StateForPlayer(peer.ID)
				stateData, _ := json.Marshal(state)
				env, _ := p2p.NewEnvelope(p2p.MsgGameState, "", hub.LocalID(), json.RawMessage(stateData))
				hub.SendTo(peer.ID, env)
			}
			return nil
		}
	}

	hub := m.hub
	return func() tea.Msg {
		env, _ := p2p.NewEnvelope(p2p.MsgGameAction, "", hub.LocalID(), p2p.GameAction{
			Action: string(action),
			Amount: amount,
		})
		hub.Send(env)
		return nil
	}
}

func (m *LinkModel) dealNextHand() tea.Cmd {
	if !m.hub.IsHost() || m.pokerGame == nil {
		return nil
	}
	m.pokerGame.StartHand()
	m.pokerState = ptrState(m.pokerGame.State())

	hub := m.hub
	game := m.pokerGame
	return func() tea.Msg {
		for _, peer := range hub.Peers() {
			state := game.StateForPlayer(peer.ID)
			stateData, _ := json.Marshal(state)
			env, _ := p2p.NewEnvelope(p2p.MsgGameState, "", hub.LocalID(), json.RawMessage(stateData))
			hub.SendTo(peer.ID, env)
		}
		return nil
	}
}

// --- Helpers ---

func (m *LinkModel) addSystemChat(text string) {
	m.chatMessages = append(m.chatMessages, chatMsg{
		Sender: "SYSTEM",
		Text:   text,
		Time:   time.Now(),
	})
	m.updateChatViewport()
}

func (m *LinkModel) updateChatViewport() {
	content := m.renderChatMessages()
	m.chatViewport.SetContent(content)
	m.chatViewport.GotoBottom()
}

func (m *LinkModel) removePeerName(name string) {
	for i, n := range m.peerNames {
		if n == name {
			m.peerNames = append(m.peerNames[:i], m.peerNames[i+1:]...)
			return
		}
	}
}

func (m LinkModel) renderChatMessages() string {
	var b strings.Builder
	for _, msg := range m.chatMessages {
		ts := msg.Time.Format("15:04")
		switch msg.Sender {
		case "SYSTEM":
			b.WriteString(theme.DimStyle.Render(fmt.Sprintf(" [%s] %s", ts, msg.Text)))
		case "You":
			b.WriteString(theme.AmberStyle.Render(fmt.Sprintf(" [%s] %s: ", ts, msg.Sender)))
			b.WriteString(theme.BaseStyle.Render(msg.Text))
		default:
			b.WriteString(theme.BoldStyle.Render(fmt.Sprintf(" [%s] %s: ", ts, msg.Sender)))
			b.WriteString(theme.BaseStyle.Render(msg.Text))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func ptrState(s games.HoldemState) *games.HoldemState {
	return &s
}

func waitForP2PMsg(hub *p2p.Hub) tea.Cmd {
	return func() tea.Msg {
		env, ok := <-hub.IncomingCh
		if !ok {
			return p2pErrorMsg{Err: fmt.Errorf("connection closed")}
		}
		return p2pIncomingMsg{Envelope: env}
	}
}

// --- Views ---

func (m *LinkModel) View(width, height int) string {
	if width != m.width || height != m.height {
		m.width = width
		m.height = height
	}

	switch m.state {
	case linkLobby:
		return m.viewLobby()
	case linkHosting:
		return m.viewHosting()
	case linkJoining:
		return m.viewJoining()
	case linkConnected:
		return m.viewConnected()
	case linkChat:
		return m.viewChat()
	case linkPoker:
		return m.viewPoker()
	}
	return ""
}

func (m *LinkModel) viewLobbyForm(title string) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" " + title + " "))
	b.WriteString("\n\n")

	b.WriteString(theme.BoldStyle.Render("  COURIER IDENTITY"))
	b.WriteString("\n\n")

	labelWidth := 16
	for i, f := range m.lobbyFields {
		cursor := "  "
		if i == m.lobbyCursor {
			cursor = theme.AmberStyle.Render("> ")
		}

		label := theme.DimStyle.Render(fmt.Sprintf("%-*s", labelWidth, f.Label+":"))

		var val string
		if m.lobbyEditing && i == m.lobbyCursor {
			val = m.input.View()
		} else {
			displayVal := f.Value
			if displayVal == "" {
				displayVal = "(empty)"
			} else if f.IsPassword {
				displayVal = strings.Repeat("•", len(displayVal))
			}
			valStyle := theme.BaseStyle
			if i == m.lobbyCursor {
				valStyle = theme.AmberStyle
			}
			if displayVal == "(empty)" {
				valStyle = theme.DimStyle
			}
			val = valStyle.Render(displayVal)
		}

		line := fmt.Sprintf("  %s%s %s", cursor, label, val)
		if !m.lobbyEditing || i != m.lobbyCursor {
			line = zone.Mark(fmt.Sprintf("link-field-%d", i), line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.errorMsg != "" {
		b.WriteString(theme.RedStyle.Render("  ERROR: " + m.errorMsg))
		b.WriteString("\n\n")
	}

	if m.statusMsg != "" {
		b.WriteString(theme.DimStyle.Render("  " + m.statusMsg))
		b.WriteString("\n\n")
	}

	return b.String()
}

func (m *LinkModel) viewLobby() string {
	var b strings.Builder

	b.WriteString(m.viewLobbyForm("P2P NETWORK LINK"))

	b.WriteString("  ")
	b.WriteString(zone.Mark("link-host", theme.AmberStyle.Render("[ HOST ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-join", theme.BaseStyle.Render("[ JOIN ]")))
	b.WriteString("\n\n")

	if m.lobbyEditing {
		b.WriteString(theme.DimStyle.Render("  [Enter] Save  [Esc] Cancel"))
	} else {
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Edit  [H] Host  [Tab] Host"))
	}

	return b.String()
}

func (m *LinkModel) viewJoining() string {
	var b strings.Builder

	b.WriteString(m.viewLobbyForm("JOIN NETWORK"))

	b.WriteString("  ")
	b.WriteString(zone.Mark("link-connect", theme.AmberStyle.Render("[ CONNECT ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-back", theme.BaseStyle.Render("[ BACK ]")))
	b.WriteString("\n\n")

	if m.lobbyEditing {
		b.WriteString(theme.DimStyle.Render("  [Enter] Save  [Esc] Cancel"))
	} else {
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Edit  [Tab] Connect  [Esc] Back"))
	}

	return b.String()
}

func (m *LinkModel) viewHosting() string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" HOSTING - WAITING FOR COURIERS "))
	b.WriteString("\n\n")

	b.WriteString(theme.AmberStyle.Render("  BROADCASTING ON: "))
	b.WriteString(theme.BoldStyle.Render(m.hostAddress))
	b.WriteString("\n\n")

	b.WriteString(theme.DimStyle.Render("  Share the address and passphrase with other Couriers."))
	b.WriteString("\n\n")

	if len(m.peerNames) > 0 {
		b.WriteString(theme.BoldStyle.Render("  CONNECTED COURIERS:"))
		b.WriteString("\n")
		for _, name := range m.peerNames {
			b.WriteString(theme.BaseStyle.Render(fmt.Sprintf("    > %s", name)))
			b.WriteString("\n")
		}
		b.WriteString("\n")

		b.WriteString("  ")
		b.WriteString(zone.Mark("link-chat", theme.AmberStyle.Render("[ CHAT ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("link-poker", theme.BaseStyle.Render("[ POKER ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("link-cancel", theme.RedStyle.Render("[ STOP ]")))
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  [C] Chat  [P] Poker  [Esc] Stop"))
	} else {
		b.WriteString(theme.DimStyle.Render("  Waiting for connections..."))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(zone.Mark("link-cancel", theme.RedStyle.Render("[ STOP ]")))
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  [Esc] Cancel"))
	}

	return b.String()
}

func (m *LinkModel) viewConnected() string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" NETWORK CONNECTED "))
	b.WriteString("\n\n")

	b.WriteString(theme.BoldStyle.Render("  CONNECTED COURIERS:"))
	b.WriteString("\n")
	for _, name := range m.peerNames {
		b.WriteString(theme.BaseStyle.Render(fmt.Sprintf("    > %s", name)))
		b.WriteString("\n")
	}
	if peers := m.hub.Peers(); len(peers) > 0 {
		for _, p := range peers {
			found := false
			for _, n := range m.peerNames {
				if n == p.Name {
					found = true
					break
				}
			}
			if !found {
				b.WriteString(theme.BaseStyle.Render(fmt.Sprintf("    > %s", p.Name)))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")

	if len(m.chatMessages) > 0 {
		b.WriteString(theme.BoldStyle.Render("  RECENT MESSAGES:"))
		b.WriteString("\n")
		start := len(m.chatMessages) - 3
		if start < 0 {
			start = 0
		}
		for _, msg := range m.chatMessages[start:] {
			ts := msg.Time.Format("15:04")
			if msg.Sender == "SYSTEM" {
				b.WriteString(theme.DimStyle.Render(fmt.Sprintf("    [%s] %s", ts, msg.Text)))
			} else {
				b.WriteString(theme.DimStyle.Render(fmt.Sprintf("    [%s] %s: %s", ts, msg.Sender, msg.Text)))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("  ")
	b.WriteString(zone.Mark("link-chat", theme.AmberStyle.Render("[ CHAT ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-poker", theme.BaseStyle.Render("[ POKER ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-disconnect", theme.RedStyle.Render("[ DISCONNECT ]")))
	b.WriteString("\n\n")

	b.WriteString(theme.DimStyle.Render("  [C] Chat  [P] Poker  [Esc] Disconnect"))

	return b.String()
}

func (m *LinkModel) viewChat() string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" COURIER RADIO CHAT "))
	b.WriteString("\n\n")

	vpHeight := m.height - 8
	if vpHeight < 5 {
		vpHeight = 5
	}
	vpWidth := m.width - 4
	if vpWidth < 20 {
		vpWidth = 20
	}

	if m.chatViewport.Width != vpWidth || m.chatViewport.Height != vpHeight {
		m.chatViewport = viewport.New(vpWidth, vpHeight)
		m.updateChatViewport()
	}

	b.WriteString(m.chatViewport.View())
	b.WriteString("\n")

	divWidth := m.width - 4
	if divWidth < 10 {
		divWidth = 10
	}
	b.WriteString(theme.DimStyle.Render(strings.Repeat("─", divWidth)))
	b.WriteString("\n")

	b.WriteString(m.chatInput.View())
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-send", theme.AmberStyle.Render("[ SEND ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("link-back-lobby", theme.BaseStyle.Render("[ BACK ]")))

	return b.String()
}

func (m *LinkModel) viewPoker() string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" CAPS CASINO - TEXAS HOLD'EM "))
	b.WriteString("\n\n")

	if m.pokerState == nil {
		b.WriteString(theme.DimStyle.Render("  Waiting for game state..."))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(zone.Mark("poker-back", theme.BaseStyle.Render("[ BACK ]")))
		return b.String()
	}

	st := m.pokerState

	// Phase + pot
	b.WriteString(theme.AmberStyle.Render(fmt.Sprintf("  Phase: %s", st.Phase)))
	b.WriteString(theme.BaseStyle.Render(fmt.Sprintf("  |  Pot: %d CAPS", st.Pot)))
	if st.CurrentBet > 0 {
		b.WriteString(theme.DimStyle.Render(fmt.Sprintf("  |  Current Bet: %d", st.CurrentBet)))
	}
	b.WriteString("\n\n")

	// Community cards
	if len(st.Community) > 0 {
		b.WriteString(theme.BoldStyle.Render("  COMMUNITY CARDS"))
		b.WriteString("\n")
		var cardRenderings [][]string
		for _, c := range st.Community {
			cardRenderings = append(cardRenderings, games.RenderCard(c))
		}
		for len(cardRenderings) < 5 {
			cardRenderings = append(cardRenderings, games.RenderCardBack())
		}
		cardStr := games.RenderCardsHorizontal(cardRenderings)
		for _, line := range strings.Split(cardStr, "\n") {
			b.WriteString(theme.BaseStyle.Render("  " + line))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Players
	b.WriteString(theme.BoldStyle.Render("  PLAYERS"))
	b.WriteString("\n")
	myID := ""
	if m.hub != nil {
		myID = m.hub.LocalID()
	}

	for i, p := range st.Players {
		marker := "  "
		if i == st.ActiveIdx && st.Phase >= games.PhasePreFlop && st.Phase <= games.PhaseRiver {
			marker = "> "
		}
		dealerMark := ""
		if i == st.DealerIdx {
			dealerMark = " [D]"
		}

		status := ""
		if p.Folded {
			status = " (FOLDED)"
		} else if p.AllIn {
			status = " (ALL-IN)"
		}

		line := fmt.Sprintf("%s%-12s %4d CAPS  bet:%3d%s%s", marker, p.Name, p.Chips, p.Bet, dealerMark, status)

		if p.ID == myID {
			b.WriteString(theme.AmberStyle.Render("  " + line))
		} else if p.Folded {
			b.WriteString(theme.DimStyle.Render("  " + line))
		} else {
			b.WriteString(theme.BaseStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Hole cards
	for _, p := range st.Players {
		if p.ID == myID && len(p.HoleCards) == 2 {
			b.WriteString(theme.BoldStyle.Render("  YOUR HAND"))
			b.WriteString("\n")
			cardRenderings := [][]string{
				games.RenderCard(p.HoleCards[0]),
				games.RenderCard(p.HoleCards[1]),
			}
			cardStr := games.RenderCardsHorizontal(cardRenderings)
			for _, line := range strings.Split(cardStr, "\n") {
				b.WriteString(theme.AmberStyle.Render("  " + line))
				b.WriteString("\n")
			}
			b.WriteString("\n")
			break
		}
	}

	// Winner display
	if st.Phase == games.PhaseHandOver && len(st.Winners) > 0 {
		winStr := strings.Join(st.Winners, ", ")
		b.WriteString(theme.AmberStyle.Render(fmt.Sprintf("  WINNER: %s — %s (%d CAPS)", winStr, st.WinHand, st.WinAmount)))
		b.WriteString("\n\n")

		for _, p := range st.Players {
			if !p.Folded && len(p.HoleCards) == 2 && p.ID != myID {
				b.WriteString(theme.DimStyle.Render(fmt.Sprintf("  %s: %s %s", p.Name, p.HoleCards[0], p.HoleCards[1])))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")

		if m.hub != nil && m.hub.IsHost() {
			b.WriteString("  ")
			b.WriteString(zone.Mark("poker-deal", theme.AmberStyle.Render("[ DEAL NEXT HAND ]")))
			b.WriteString("  ")
		}
		b.WriteString(zone.Mark("poker-back", theme.BaseStyle.Render("[ LEAVE TABLE ]")))
		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render("  [Enter] Deal  [Esc] Leave"))
		return b.String()
	}

	// Error
	if m.errorMsg != "" {
		b.WriteString(theme.RedStyle.Render("  " + m.errorMsg))
		b.WriteString("\n\n")
	}

	// Action buttons
	isMyTurn := false
	if st.Phase >= games.PhasePreFlop && st.Phase <= games.PhaseRiver {
		if st.ActiveIdx >= 0 && st.ActiveIdx < len(st.Players) {
			isMyTurn = st.Players[st.ActiveIdx].ID == myID
		}
	}

	if isMyTurn {
		toCall := st.CurrentBet
		for _, p := range st.Players {
			if p.ID == myID {
				toCall = st.CurrentBet - p.Bet
				break
			}
		}

		b.WriteString("  ")
		b.WriteString(zone.Mark("poker-fold", theme.RedStyle.Render("[ FOLD ]")))
		b.WriteString("  ")
		if toCall <= 0 {
			b.WriteString(zone.Mark("poker-check", theme.BaseStyle.Render("[ CHECK ]")))
		} else {
			b.WriteString(zone.Mark("poker-call", theme.BaseStyle.Render(fmt.Sprintf("[ CALL %d ]", toCall))))
		}
		b.WriteString("  ")
		b.WriteString(zone.Mark("poker-allin", theme.AmberStyle.Render("[ ALL-IN ]")))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(m.raiseInput.View())
		b.WriteString(" ")
		b.WriteString(zone.Mark("poker-raise", theme.BaseStyle.Render("[ RAISE ]")))
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  [F]old [K]Check [C]all [R]aise [A]ll-in"))
	} else {
		if st.Phase >= games.PhasePreFlop && st.Phase <= games.PhaseRiver {
			b.WriteString(theme.DimStyle.Render("  Waiting for other player..."))
		}
	}
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("poker-back", theme.BaseStyle.Render("[ LEAVE TABLE ]")))

	return b.String()
}
