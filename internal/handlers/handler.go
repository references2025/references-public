package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"references/internal/game"
	"references/internal/utils"
	"strconv"
	"time"
)

func marshalToJS(v interface{}) (template.JS, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data to JSON: %w", err)
	}
	return template.JS(jsonData), nil
}

type Handlers struct {
	game *game.Game
}

func NewHandlers(g *game.Game) *Handlers {
	return &Handlers{game: g}
}

func parseTemplate(filenames ...string) (*template.Template, error) {
	return template.ParseFiles(filenames...)
}

func (h *Handlers) IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := parseTemplate("web/templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing index.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	categoryEmojisJS, err := marshalToJS(h.game.GetAllCategoryEmojis())
	if err != nil {
		fmt.Printf("Error marshalling emojis for index: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	gameIDJS, err := marshalToJS(h.game.GetDailyGameID())
	if err != nil {
		fmt.Printf("Error marshalling gameID for index: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	baseURLJS, err := marshalToJS(h.game.Cfg.BaseGameURL)
	if err != nil {
		fmt.Printf("Error marshalling baseURL for index: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		MaskedWord     string
		Categories     []string
		CategoryEmojis template.JS
		GameID         template.JS
		BaseGameURL    template.JS
	}{
		MaskedWord:     h.game.GetMaskedWord(),
		Categories:     h.game.GetCategories(),
		CategoryEmojis: categoryEmojisJS,
		GameID:         gameIDJS,
		BaseGameURL:    baseURLJS,
	}

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Printf("Error executing index.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) SuccessHandler(w http.ResponseWriter, r *http.Request) {
	baseDate := time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC)
	today := time.Now()
	gameNumber := int(today.Sub(baseDate).Hours()/24) + 1
	formattedDate := today.Format("2-Jan-2006")

	tmpl, err := parseTemplate("web/templates/success.html")
	if err != nil {
		fmt.Printf("Error parsing success.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	guessesStr := r.URL.Query().Get("guesses")
	guesses, _ := strconv.Atoi(guessesStr)
	if guesses <= 0 {
		guesses = 1
	}
	hintsStr := r.URL.Query().Get("hints")
	hints, _ := strconv.Atoi(hintsStr)
	word := r.URL.Query().Get("word")
	gameID := r.URL.Query().Get("gameId")

	categoryEmojisJS, err := marshalToJS(h.game.GetAllCategoryEmojis())
	if err != nil {
		fmt.Printf("Error marshalling emojis for success: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	gameIDJS, err := marshalToJS(gameID)
	if err != nil {
		fmt.Printf("Error marshalling gameID for success: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	wordJS, err := marshalToJS(word)
	if err != nil {
		fmt.Printf("Error marshalling word for success: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	baseURLJS, err := marshalToJS(h.game.Cfg.BaseGameURL)
	if err != nil {
		fmt.Printf("Error marshalling baseURL for success: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Word          string
		Guesses       int
		Hints         int
		GameIDDisplay string
		GameNumber    int
		FormattedDate string

		WordJS         template.JS
		GameIDJS       template.JS
		CategoryEmojis template.JS
		BaseGameURLJS  template.JS
	}{
		Word:          word,
		Guesses:       guesses,
		Hints:         hints,
		GameIDDisplay: gameID,
		GameNumber:    gameNumber,
		FormattedDate: formattedDate,

		WordJS:         wordJS,
		GameIDJS:       gameIDJS,
		CategoryEmojis: categoryEmojisJS,
		BaseGameURLJS:  baseURLJS,
	}

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Printf("Error executing success.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) MaybeTomorrowHandler(w http.ResponseWriter, r *http.Request) {
	baseDate := time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC)
	today := time.Now()
	gameNumber := int(today.Sub(baseDate).Hours()/24) + 1
	formattedDate := today.Format("2-Jan-2006")

	tmpl, err := parseTemplate("web/templates/maybe-tomorrow.html")
	if err != nil {
		fmt.Printf("Error parsing maybe-tomorrow.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	word := r.URL.Query().Get("word")
	gameID := r.URL.Query().Get("gameId")

	guessesStr := r.URL.Query().Get("guesses")
	guesses, err := strconv.Atoi(guessesStr)
	if err != nil || guesses <= 0 {
		guesses = 4
	}

	hintsStr := r.URL.Query().Get("hints")
	hints, err := strconv.Atoi(hintsStr)
	if err != nil || hints < 0 {
		hints = 0
	}

	categoryEmojisJS, err := marshalToJS(h.game.GetAllCategoryEmojis())
	if err != nil {
		fmt.Printf("Error marshalling emojis for tomorrow: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	gameIDJS, err := marshalToJS(gameID)
	if err != nil {
		fmt.Printf("Error marshalling gameID for tomorrow: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	wordJS, err := marshalToJS(word)
	if err != nil {
		fmt.Printf("Error marshalling word for tomorrow: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	baseURLJS, err := marshalToJS(h.game.Cfg.BaseGameURL)
	if err != nil {
		fmt.Printf("Error marshalling baseURL for tomorrow: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Word           string
		GameIDDisplay  string
		Guesses        int
		Hints          int
		GameNumber     int
		FormattedDate  string
		WordJS         template.JS
		GameIDJS       template.JS
		CategoryEmojis template.JS
		BaseGameURLJS  template.JS
	}{
		Word:           word,
		GameIDDisplay:  gameID,
		GameNumber:     gameNumber,
		FormattedDate:  formattedDate,
		Guesses:        guesses,
		Hints:          hints,
		WordJS:         wordJS,
		GameIDJS:       gameIDJS,
		CategoryEmojis: categoryEmojisJS,
		BaseGameURLJS:  baseURLJS,
	}

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Printf("Error executing maybe-tomorrow.html: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) GuessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Error parsing form")
		return
	}

	guess := r.FormValue("guess")
	playerID := r.FormValue("playerID")

	if guess == "" {
		utils.RespondError(w, http.StatusBadRequest, "Guess cannot be empty")
		return
	}

	correct, revealedPositions := h.game.CheckGuess(guess)

	response := struct {
		Correct           bool   `json:"correct"`
		Word              string `json:"word"`
		MaskedWord        string `json:"maskedWord"`
		RevealedPositions []int  `json:"revealedPositions,omitempty"`
	}{
		Correct:    correct,
		Word:       h.game.Word,
		MaskedWord: h.game.GetPartiallyRevealedWord(),
	}
	if game.EnablePartialUnmasking && len(revealedPositions) > 0 {
		response.RevealedPositions = revealedPositions
	}

	utils.RespondJSON(w, http.StatusOK, response)

	h.game.Sheet.LogEvent(game.Event{
		GameID:    h.game.GetDailyGameID(),
		PlayerID:  playerID,
		EventType: "guess",
		Data: map[string]string{
			"guess":   guess,
			"correct": strconv.FormatBool(correct),
		},
		Timestamp: time.Now(),
	})
}

func (h *Handlers) HintHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Error parsing form")
		return
	}

	category := r.FormValue("category")
	playerID := r.FormValue("playerID")

	if category == "" {
		utils.RespondError(w, http.StatusBadRequest, "Category cannot be empty")
		return
	}

	hint, err := h.game.GetHint(category)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	emoji, err := h.game.GetEmoji(category)
	if err != nil {
		fmt.Printf("Warning: Could not get emoji for category '%s': %v\n", category, err)
		utils.RespondError(w, http.StatusInternalServerError, "Could not retrieve hint details")
		return
	}

	response := struct {
		Hint  string `json:"hint"`
		Emoji string `json:"emoji"`
	}{
		Hint:  hint,
		Emoji: emoji,
	}

	utils.RespondJSON(w, http.StatusOK, response)

	h.game.Sheet.LogEvent(game.Event{
		GameID:    h.game.GetDailyGameID(),
		PlayerID:  playerID,
		EventType: "hint",
		Data: map[string]string{
			"category": category,
		},
		Timestamp: time.Now(),
	})
}

func (h *Handlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	gameID := r.URL.Query().Get("gameId")
	playerID := r.URL.Query().Get("playerId")

	if gameID == "" || playerID == "" {
		utils.RespondError(w, http.StatusBadRequest, "Game ID and Player ID are required")
		return
	}

	stats, err := h.game.Sheet.GetPlayerStats(gameID, playerID)

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, stats)
}
