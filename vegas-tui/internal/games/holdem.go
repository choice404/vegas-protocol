package games

import (
	"fmt"
	"sort"
)

// GamePhase represents the current phase of a Hold'em hand.
type GamePhase int

const (
	PhaseWaiting GamePhase = iota
	PhasePreFlop
	PhaseFlop
	PhaseTurn
	PhaseRiver
	PhaseShowdown
	PhaseHandOver
)

var phaseNames = [7]string{
	"Waiting", "Pre-Flop", "Flop", "Turn", "River", "Showdown", "Hand Over",
}

func (p GamePhase) String() string { return phaseNames[p] }

// PlayerAction represents a betting action.
type PlayerAction string

const (
	ActionFold  PlayerAction = "fold"
	ActionCheck PlayerAction = "check"
	ActionCall  PlayerAction = "call"
	ActionRaise PlayerAction = "raise"
	ActionAllIn PlayerAction = "allin"
)

// Player represents a player in a Hold'em game.
type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Chips    int    `json:"chips"`
	Bet      int    `json:"bet"`
	Folded   bool   `json:"folded"`
	AllIn    bool   `json:"all_in"`
	HoleCards []Card `json:"hole_cards,omitempty"`
}

// HoldemState is the serializable game state sent to clients.
type HoldemState struct {
	Phase      GamePhase `json:"phase"`
	Players    []Player  `json:"players"`
	Community  []Card    `json:"community"`
	Pot        int       `json:"pot"`
	CurrentBet int       `json:"current_bet"`
	ActiveIdx  int       `json:"active_idx"`
	DealerIdx  int       `json:"dealer_idx"`
	SmallBlind int       `json:"small_blind"`
	BigBlind   int       `json:"big_blind"`
	Winners    []string  `json:"winners,omitempty"`
	WinHand    string    `json:"win_hand,omitempty"`
	WinAmount  int       `json:"win_amount,omitempty"`
}

// HoldemGame is the authoritative game engine (host-only).
type HoldemGame struct {
	state     HoldemState
	deck      *Deck
	lastRaise int
	actedThisRound map[string]bool
}

// NewHoldemGame creates a new game with the given players and starting chips.
func NewHoldemGame(players []Player, smallBlind, bigBlind int) *HoldemGame {
	for i := range players {
		if players[i].Chips == 0 {
			players[i].Chips = 1000
		}
	}
	g := &HoldemGame{
		state: HoldemState{
			Phase:      PhaseWaiting,
			Players:    players,
			SmallBlind: smallBlind,
			BigBlind:   bigBlind,
			DealerIdx:  0,
		},
		actedThisRound: make(map[string]bool),
	}
	return g
}

// AddPlayer adds a player to the game (only in Waiting/HandOver phase).
func (g *HoldemGame) AddPlayer(p Player) error {
	if g.state.Phase != PhaseWaiting && g.state.Phase != PhaseHandOver {
		return fmt.Errorf("cannot add player during active hand")
	}
	if len(g.state.Players) >= 6 {
		return fmt.Errorf("maximum 6 players")
	}
	if p.Chips == 0 {
		p.Chips = 1000
	}
	g.state.Players = append(g.state.Players, p)
	return nil
}

// StartHand begins a new hand: shuffles, posts blinds, deals hole cards.
func (g *HoldemGame) StartHand() error {
	if len(g.state.Players) < 2 {
		return fmt.Errorf("need at least 2 players")
	}

	// Reset player state
	for i := range g.state.Players {
		g.state.Players[i].Bet = 0
		g.state.Players[i].Folded = false
		g.state.Players[i].AllIn = false
		g.state.Players[i].HoleCards = nil
	}
	g.state.Community = nil
	g.state.Pot = 0
	g.state.CurrentBet = 0
	g.state.Winners = nil
	g.state.WinHand = ""
	g.state.WinAmount = 0
	g.actedThisRound = make(map[string]bool)

	// Advance dealer
	g.state.DealerIdx = (g.state.DealerIdx + 1) % len(g.state.Players)

	// Shuffle deck
	g.deck = NewDeck()
	g.deck.Shuffle()

	n := len(g.state.Players)
	sbIdx := (g.state.DealerIdx + 1) % n
	bbIdx := (g.state.DealerIdx + 2) % n

	// Post blinds
	g.postBlind(sbIdx, g.state.SmallBlind)
	g.postBlind(bbIdx, g.state.BigBlind)
	g.state.CurrentBet = g.state.BigBlind

	// Deal 2 hole cards to each player
	for i := range g.state.Players {
		g.state.Players[i].HoleCards = g.deck.DealN(2)
	}

	g.state.Phase = PhasePreFlop
	// First to act is after big blind
	g.state.ActiveIdx = (bbIdx + 1) % n
	g.skipFolded()
	g.lastRaise = g.state.BigBlind

	return nil
}

func (g *HoldemGame) postBlind(idx, amount int) {
	p := &g.state.Players[idx]
	if p.Chips <= amount {
		amount = p.Chips
		p.AllIn = true
	}
	p.Chips -= amount
	p.Bet = amount
	g.state.Pot += amount
}

// ProcessAction handles a player's action. Returns an error if invalid.
func (g *HoldemGame) ProcessAction(playerID string, action PlayerAction, raiseAmount int) error {
	if g.state.Phase < PhasePreFlop || g.state.Phase > PhaseRiver {
		return fmt.Errorf("no active betting round")
	}

	p := &g.state.Players[g.state.ActiveIdx]
	if p.ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if p.Folded || p.AllIn {
		return fmt.Errorf("cannot act (folded or all-in)")
	}

	toCall := g.state.CurrentBet - p.Bet

	switch action {
	case ActionFold:
		p.Folded = true

	case ActionCheck:
		if toCall > 0 {
			return fmt.Errorf("cannot check, must call %d", toCall)
		}

	case ActionCall:
		if toCall <= 0 {
			return fmt.Errorf("nothing to call, use check")
		}
		if p.Chips <= toCall {
			// All-in call
			g.state.Pot += p.Chips
			p.Bet += p.Chips
			p.Chips = 0
			p.AllIn = true
		} else {
			p.Chips -= toCall
			p.Bet += toCall
			g.state.Pot += toCall
		}

	case ActionRaise:
		minRaise := g.state.CurrentBet + g.lastRaise
		if raiseAmount < minRaise && p.Chips+p.Bet > minRaise {
			return fmt.Errorf("minimum raise to %d", minRaise)
		}
		totalNeeded := raiseAmount - p.Bet
		if totalNeeded >= p.Chips {
			// All-in raise
			g.state.Pot += p.Chips
			raiseAmount = p.Bet + p.Chips
			p.Bet = raiseAmount
			p.Chips = 0
			p.AllIn = true
		} else {
			p.Chips -= totalNeeded
			g.state.Pot += totalNeeded
			p.Bet = raiseAmount
		}
		g.lastRaise = raiseAmount - g.state.CurrentBet
		g.state.CurrentBet = raiseAmount
		// Reset acted flags since there's a new bet to respond to
		g.actedThisRound = map[string]bool{p.ID: true}

	case ActionAllIn:
		allInAmount := p.Chips + p.Bet
		g.state.Pot += p.Chips
		if allInAmount > g.state.CurrentBet {
			g.lastRaise = allInAmount - g.state.CurrentBet
			g.state.CurrentBet = allInAmount
			g.actedThisRound = map[string]bool{p.ID: true}
		}
		p.Bet = allInAmount
		p.Chips = 0
		p.AllIn = true

	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	g.actedThisRound[p.ID] = true

	// Check if hand is over (only one player left)
	activePlayers := g.countActive()
	if activePlayers <= 1 {
		g.resolveLastPlayer()
		return nil
	}

	// Advance to next player
	g.advancePlayer()

	// Check if betting round is complete
	if g.isBettingComplete() {
		g.advancePhase()
	}

	return nil
}

func (g *HoldemGame) countActive() int {
	count := 0
	for _, p := range g.state.Players {
		if !p.Folded {
			count++
		}
	}
	return count
}

func (g *HoldemGame) countCanAct() int {
	count := 0
	for _, p := range g.state.Players {
		if !p.Folded && !p.AllIn {
			count++
		}
	}
	return count
}

func (g *HoldemGame) resolveLastPlayer() {
	for _, p := range g.state.Players {
		if !p.Folded {
			g.state.Winners = []string{p.Name}
			g.state.WinAmount = g.state.Pot
			g.state.WinHand = "Last player standing"
			// Award pot
			for i := range g.state.Players {
				if g.state.Players[i].ID == p.ID {
					g.state.Players[i].Chips += g.state.Pot
				}
			}
			break
		}
	}
	g.state.Phase = PhaseHandOver
}

func (g *HoldemGame) advancePlayer() {
	n := len(g.state.Players)
	for i := 0; i < n; i++ {
		g.state.ActiveIdx = (g.state.ActiveIdx + 1) % n
		p := g.state.Players[g.state.ActiveIdx]
		if !p.Folded && !p.AllIn {
			return
		}
	}
}

func (g *HoldemGame) skipFolded() {
	n := len(g.state.Players)
	for i := 0; i < n; i++ {
		p := g.state.Players[g.state.ActiveIdx]
		if !p.Folded && !p.AllIn {
			return
		}
		g.state.ActiveIdx = (g.state.ActiveIdx + 1) % n
	}
}

func (g *HoldemGame) isBettingComplete() bool {
	for _, p := range g.state.Players {
		if p.Folded || p.AllIn {
			continue
		}
		if !g.actedThisRound[p.ID] {
			return false
		}
		if p.Bet != g.state.CurrentBet {
			return false
		}
	}
	return true
}

func (g *HoldemGame) advancePhase() {
	// Reset bets for new round
	for i := range g.state.Players {
		g.state.Players[i].Bet = 0
	}
	g.state.CurrentBet = 0
	g.lastRaise = g.state.BigBlind
	g.actedThisRound = make(map[string]bool)

	canAct := g.countCanAct()

	switch g.state.Phase {
	case PhasePreFlop:
		g.state.Phase = PhaseFlop
		g.state.Community = append(g.state.Community, g.deck.DealN(3)...)
	case PhaseFlop:
		g.state.Phase = PhaseTurn
		g.state.Community = append(g.state.Community, g.deck.Deal())
	case PhaseTurn:
		g.state.Phase = PhaseRiver
		g.state.Community = append(g.state.Community, g.deck.Deal())
	case PhaseRiver:
		g.resolveShowdown()
		return
	}

	// If no one can act, fast-forward through remaining community cards
	if canAct <= 1 {
		for g.state.Phase < PhaseRiver {
			switch g.state.Phase {
			case PhaseFlop:
				g.state.Phase = PhaseTurn
				g.state.Community = append(g.state.Community, g.deck.Deal())
			case PhaseTurn:
				g.state.Phase = PhaseRiver
				g.state.Community = append(g.state.Community, g.deck.Deal())
			}
		}
		g.resolveShowdown()
		return
	}

	// Set first actor to left of dealer
	n := len(g.state.Players)
	g.state.ActiveIdx = (g.state.DealerIdx + 1) % n
	g.skipFolded()
}

func (g *HoldemGame) resolveShowdown() {
	g.state.Phase = PhaseShowdown

	type candidate struct {
		idx    int
		result EvalResult
	}

	var candidates []candidate
	for i, p := range g.state.Players {
		if p.Folded {
			continue
		}
		allCards := append([]Card{}, p.HoleCards...)
		allCards = append(allCards, g.state.Community...)
		result := EvaluateBestHand(allCards)
		candidates = append(candidates, candidate{idx: i, result: result})
	}

	// Sort by hand strength descending
	sort.Slice(candidates, func(i, j int) bool {
		return CompareHands(candidates[i].result, candidates[j].result) > 0
	})

	// Find all winners (ties)
	var winners []int
	bestResult := candidates[0].result
	for _, c := range candidates {
		if CompareHands(c.result, bestResult) == 0 {
			winners = append(winners, c.idx)
		} else {
			break
		}
	}

	share := g.state.Pot / len(winners)
	remainder := g.state.Pot % len(winners)

	var winnerNames []string
	for i, idx := range winners {
		g.state.Players[idx].Chips += share
		if i == 0 {
			g.state.Players[idx].Chips += remainder
		}
		winnerNames = append(winnerNames, g.state.Players[idx].Name)
	}

	g.state.Winners = winnerNames
	g.state.WinHand = bestResult.Rank.String()
	g.state.WinAmount = g.state.Pot
	g.state.Phase = PhaseHandOver
}

// State returns a copy of the full game state (host view with all cards visible).
func (g *HoldemGame) State() HoldemState {
	return g.state
}

// StateForPlayer returns a state copy where other players' hole cards are hidden.
func (g *HoldemGame) StateForPlayer(playerID string) HoldemState {
	s := g.state
	s.Players = make([]Player, len(g.state.Players))
	copy(s.Players, g.state.Players)

	for i := range s.Players {
		if s.Players[i].ID != playerID {
			// Hide hole cards unless showdown
			if s.Phase != PhaseShowdown && s.Phase != PhaseHandOver {
				s.Players[i].HoleCards = nil
			}
		}
	}
	return s
}

// PlayerIndex returns the index of a player by ID, or -1.
func (g *HoldemGame) PlayerIndex(playerID string) int {
	for i, p := range g.state.Players {
		if p.ID == playerID {
			return i
		}
	}
	return -1
}

// RemovePlayer removes a player from the game.
func (g *HoldemGame) RemovePlayer(playerID string) {
	for i, p := range g.state.Players {
		if p.ID == playerID {
			g.state.Players[i].Folded = true
			if g.countActive() <= 1 && g.state.Phase >= PhasePreFlop && g.state.Phase <= PhaseRiver {
				g.resolveLastPlayer()
			}
			return
		}
	}
}
