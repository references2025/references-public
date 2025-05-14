package game

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"references/internal/config"
)

const (
	EnablePartialUnmasking = false

	UseSequentialDailyWord = true

	DailyWordStartDateString = "2025-04-10"
)

var (
	dailyWordStartDate time.Time
)

func init() {
	var err error
	dailyWordStartDate, err = time.Parse("2006-01-02", DailyWordStartDateString)
	if err != nil {
		panic(fmt.Sprintf("invalid DailyWordStartDateString: %v", err))
	}
}

type Game struct {
	Cfg               config.Config
	Word              string
	RevealedPositions map[int]bool
	Hints             map[string]string
	Categories        map[string]string
	CategoryEmojis    map[string]string
	CategoryOrder     []string
	Sheet             *Sheet
}

func NewGame(cfg config.Config) (*Game, error) {
	sheet, err := NewSheet(cfg)
	if err != nil {
		return nil, err
	}

	g := &Game{
		Cfg:               cfg,
		Sheet:             sheet,
		RevealedPositions: make(map[int]bool),
		Hints:             make(map[string]string),
		Categories:        make(map[string]string),
		CategoryEmojis:    make(map[string]string),
	}

	if err := g.loadDailyWord(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Game) loadDailyWord() error {
	data, err := g.Sheet.GetDailyWord()
	if err != nil {
		return err
	}
	g.Word = data.Answer
	g.Hints = data.Hints
	g.Categories = data.Categories
	g.CategoryEmojis = data.CategoryEmojis
	g.CategoryOrder = data.CategoryOrder
	g.RevealedPositions = make(map[int]bool)
	return nil
}

func (g *Game) CheckGuess(guess string) (bool, []int) {
	normalisedGuess := strings.ToLower(strings.TrimSpace(guess))
	normalisedWord := strings.ToLower(g.Word)
	if normalisedGuess == normalisedWord {
		for i := range []rune(g.Word) {
			g.RevealedPositions[i] = true
		}
		return true, nil
	}

	var revealed []int
	if EnablePartialUnmasking {
		wr, gr := []rune(normalisedWord), []rune(normalisedGuess)
		max := len(wr)
		if len(gr) < max {
			max = len(gr)
		}
		for i := 0; i < max; i++ {
			if gr[i] == wr[i] && !g.RevealedPositions[i] {
				g.RevealedPositions[i] = true
				revealed = append(revealed, i)
			}
		}
	}
	return false, revealed
}

func (g *Game) GetMaskedWord() string {
	if len(g.RevealedPositions) == 0 {
		masked := make([]rune, utf8.RuneCountInString(g.Word))
		for i := range masked {
			masked[i] = '_'
		}
		return string(masked)
	}
	return g.GetPartiallyRevealedWord()
}

func (g *Game) GetPartiallyRevealedWord() string {
	wr := []rune(g.Word)
	out := make([]rune, len(wr))
	for i := range out {
		out[i] = '_'
	}
	for i := range g.RevealedPositions {
		out[i] = wr[i]
	}
	return string(out)
}

func (g *Game) GetHint(cat string) (string, error) {
	if h, ok := g.Hints[cat]; ok {
		return h, nil
	}
	return "", fmt.Errorf("invalid category: %s", cat)
}
func (g *Game) GetEmoji(cat string) (string, error) {
	if e, ok := g.CategoryEmojis[cat]; ok {
		return e, nil
	}
	return "", fmt.Errorf("no emoji for category: %s", cat)
}
func (g *Game) GetCategories() []string { return g.CategoryOrder }
func (g *Game) GetAllCategoryEmojis() map[string]string {
	copy := make(map[string]string, len(g.CategoryEmojis))
	for k, v := range g.CategoryEmojis {
		copy[k] = v
	}
	return copy
}
func (g *Game) GetDailyGameID() string { return time.Now().UTC().Format("2006-01-02") }
