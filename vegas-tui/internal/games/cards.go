package games

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Suit represents a card suit.
type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

var suitSymbols = [4]string{"♠", "♥", "♦", "♣"}
var suitNames = [4]string{"Spades", "Hearts", "Diamonds", "Clubs"}

func (s Suit) String() string { return suitSymbols[s] }

// Rank represents a card rank (2-14, where 14 = Ace).
type Rank int

const (
	Two   Rank = 2
	Three Rank = 3
	Four  Rank = 4
	Five  Rank = 5
	Six   Rank = 6
	Seven Rank = 7
	Eight Rank = 8
	Nine  Rank = 9
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
	Ace   Rank = 14
)

var rankSymbols = map[Rank]string{
	2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8",
	9: "9", 10: "10", 11: "J", 12: "Q", 13: "K", 14: "A",
}

func (r Rank) String() string { return rankSymbols[r] }

// Card represents a playing card.
type Card struct {
	Rank Rank
	Suit Suit
}

func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank, c.Suit)
}

// Deck is a collection of cards.
type Deck struct {
	Cards []Card
}

// NewDeck creates a standard 52-card deck.
func NewDeck() *Deck {
	d := &Deck{Cards: make([]Card, 0, 52)}
	for s := Spades; s <= Clubs; s++ {
		for r := Two; r <= Ace; r++ {
			d.Cards = append(d.Cards, Card{Rank: r, Suit: s})
		}
	}
	return d
}

// Shuffle randomizes the deck using crypto/rand (Fisher-Yates).
func (d *Deck) Shuffle() {
	for i := len(d.Cards) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		d.Cards[i], d.Cards[j.Int64()] = d.Cards[j.Int64()], d.Cards[i]
	}
}

// Deal removes and returns the top card.
func (d *Deck) Deal() Card {
	c := d.Cards[0]
	d.Cards = d.Cards[1:]
	return c
}

// DealN removes and returns n cards from the top.
func (d *Deck) DealN(n int) []Card {
	cards := make([]Card, n)
	copy(cards, d.Cards[:n])
	d.Cards = d.Cards[n:]
	return cards
}

// --- ASCII Card Rendering ---

// RenderCard returns a 5-line ASCII art representation of a card.
func RenderCard(c Card) []string {
	r := c.Rank.String()
	s := c.Suit.String()
	pad := " "
	if r == "10" {
		pad = ""
	}
	return []string{
		"┌─────┐",
		fmt.Sprintf("│%s%s   │", r, pad),
		fmt.Sprintf("│  %s  │", s),
		fmt.Sprintf("│   %s%s│", pad, r),
		"└─────┘",
	}
}

// RenderCardBack returns a face-down card.
func RenderCardBack() []string {
	return []string{
		"┌─────┐",
		"│░░░░░│",
		"│░░░░░│",
		"│░░░░░│",
		"└─────┘",
	}
}

// RenderCardsHorizontal joins multiple card renderings side-by-side.
func RenderCardsHorizontal(cards [][]string) string {
	if len(cards) == 0 {
		return ""
	}
	lines := make([]string, 5)
	for row := 0; row < 5; row++ {
		parts := make([]string, len(cards))
		for i, card := range cards {
			if row < len(card) {
				parts[i] = card[row]
			}
		}
		lines[row] = strings.Join(parts, " ")
	}
	return strings.Join(lines, "\n")
}

// --- Hand Evaluation ---

// HandRank represents the rank of a poker hand.
type HandRank int

const (
	HighCard HandRank = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
)

var handRankNames = [10]string{
	"High Card", "One Pair", "Two Pair", "Three of a Kind",
	"Straight", "Flush", "Full House", "Four of a Kind",
	"Straight Flush", "Royal Flush",
}

func (h HandRank) String() string { return handRankNames[h] }

// EvalResult is the result of evaluating a 5-card hand.
type EvalResult struct {
	Rank    HandRank
	Kickers [5]Rank // for tiebreaking, most significant first
}

// EvaluateBestHand finds the best 5-card hand from up to 7 cards (C(7,5) = 21 combos).
func EvaluateBestHand(cards []Card) EvalResult {
	n := len(cards)
	if n < 5 {
		return EvalResult{}
	}

	var best EvalResult
	first := true

	// Generate all C(n,5) combinations
	for i := 0; i < n-4; i++ {
		for j := i + 1; j < n-3; j++ {
			for k := j + 1; k < n-2; k++ {
				for l := k + 1; l < n-1; l++ {
					for m := l + 1; m < n; m++ {
						hand := [5]Card{cards[i], cards[j], cards[k], cards[l], cards[m]}
						result := evaluateFive(hand)
						if first || compareResults(result, best) > 0 {
							best = result
							first = false
						}
					}
				}
			}
		}
	}
	return best
}

// CompareHands compares two hand evaluation results. Returns >0 if a wins, <0 if b wins, 0 if tie.
func CompareHands(a, b EvalResult) int {
	return compareResults(a, b)
}

func compareResults(a, b EvalResult) int {
	if a.Rank != b.Rank {
		return int(a.Rank) - int(b.Rank)
	}
	for i := 0; i < 5; i++ {
		if a.Kickers[i] != b.Kickers[i] {
			return int(a.Kickers[i]) - int(b.Kickers[i])
		}
	}
	return 0
}

func evaluateFive(hand [5]Card) EvalResult {
	// Sort by rank descending
	sort.Slice(hand[:], func(i, j int) bool {
		return hand[i].Rank > hand[j].Rank
	})

	flush := hand[0].Suit == hand[1].Suit &&
		hand[1].Suit == hand[2].Suit &&
		hand[2].Suit == hand[3].Suit &&
		hand[3].Suit == hand[4].Suit

	straight, highCard := checkStraight(hand)

	// Count ranks
	rankCount := make(map[Rank]int)
	for _, c := range hand {
		rankCount[c.Rank]++
	}

	// Categorize by count
	var fours, threes, pairs, singles []Rank
	for r, cnt := range rankCount {
		switch cnt {
		case 4:
			fours = append(fours, r)
		case 3:
			threes = append(threes, r)
		case 2:
			pairs = append(pairs, r)
		case 1:
			singles = append(singles, r)
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i] > pairs[j] })
	sort.Slice(singles, func(i, j int) bool { return singles[i] > singles[j] })

	if straight && flush {
		if highCard == Ace {
			return EvalResult{Rank: RoyalFlush, Kickers: [5]Rank{Ace}}
		}
		return EvalResult{Rank: StraightFlush, Kickers: [5]Rank{highCard}}
	}

	if len(fours) == 1 {
		return EvalResult{Rank: FourOfAKind, Kickers: [5]Rank{fours[0], singles[0]}}
	}

	if len(threes) == 1 && len(pairs) == 1 {
		return EvalResult{Rank: FullHouse, Kickers: [5]Rank{threes[0], pairs[0]}}
	}

	if flush {
		return EvalResult{Rank: Flush, Kickers: [5]Rank{hand[0].Rank, hand[1].Rank, hand[2].Rank, hand[3].Rank, hand[4].Rank}}
	}

	if straight {
		return EvalResult{Rank: Straight, Kickers: [5]Rank{highCard}}
	}

	if len(threes) == 1 {
		return EvalResult{Rank: ThreeOfAKind, Kickers: [5]Rank{threes[0], singles[0], singles[1]}}
	}

	if len(pairs) == 2 {
		return EvalResult{Rank: TwoPair, Kickers: [5]Rank{pairs[0], pairs[1], singles[0]}}
	}

	if len(pairs) == 1 {
		return EvalResult{Rank: OnePair, Kickers: [5]Rank{pairs[0], singles[0], singles[1], singles[2]}}
	}

	return EvalResult{Rank: HighCard, Kickers: [5]Rank{hand[0].Rank, hand[1].Rank, hand[2].Rank, hand[3].Rank, hand[4].Rank}}
}

// checkStraight detects a straight (including A-2-3-4-5 wheel).
// hand must be sorted descending by rank.
func checkStraight(hand [5]Card) (bool, Rank) {
	// Normal straight check
	isStraight := true
	for i := 0; i < 4; i++ {
		if hand[i].Rank-hand[i+1].Rank != 1 {
			isStraight = false
			break
		}
	}
	if isStraight {
		return true, hand[0].Rank
	}

	// Wheel: A-2-3-4-5 (sorted: A,5,4,3,2)
	if hand[0].Rank == Ace && hand[1].Rank == Five &&
		hand[2].Rank == Four && hand[3].Rank == Three &&
		hand[4].Rank == Two {
		return true, Five // 5-high straight
	}

	return false, 0
}
