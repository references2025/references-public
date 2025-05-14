package game

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"references/internal/config"
)

type Sheet struct {
	prod      bool
	service   *sheets.Service
	sheetID   string
	static    [][]string
	analytics *Analytics
}
type PlayerStats struct {
	TotalPlayers  int    `json:"totalPlayers"`
	PlayersSolved int    `json:"playersSolved"`
	PlayerRank    int    `json:"playerRank"`
	SolveTime     string `json:"solveTime,omitempty"`
}

type WordData struct {
	Answer         string
	Hints          map[string]string
	Categories     map[string]string
	CategoryEmojis map[string]string
	CategoryOrder  []string
}

type Analytics struct {
	service    *sheets.Service
	sheetID    string
	eventsChan chan Event
	sheetCache map[string]struct{}
}

type Event struct {
	GameID    string
	PlayerID  string
	EventType string
	Data      map[string]string
	Timestamp time.Time
}

func NewSheet(cfg config.Config) (*Sheet, error) {
	if cfg.Mode == config.ModeLocal {
		rows, err := csv.NewReader(strings.NewReader(staticCSV)).ReadAll()
		if err != nil {
			return nil, err
		}
		return &Sheet{prod: false, static: rows}, nil
	}

	ctx := context.Background()
	svc, err := sheets.NewService(ctx, option.WithCredentialsFile(cfg.CredentialsJSONPath))
	if err != nil {
		return nil, err
	}
	s := &Sheet{
		prod:    true,
		service: svc,
		sheetID: cfg.WordSheetID,
	}
	s.InitAnalytics(cfg.AnalyticsSheetID)
	return s, nil
}

func (s *Sheet) GetDailyWord() (*WordData, error) {
	if !s.prod {
		return parseWordData(interfaceSlice(s.static[s.randomIndex()]))
	}

	readRange := "Sheet1!A2:M"
	resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("read prod sheet: %w", err)
	}
	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no rows in prod sheet")
	}

	idx := s.randomIndexWithLen(len(resp.Values))
	return parseWordData(resp.Values[idx])
}

func (s *Sheet) randomIndex() int { return s.randomIndexWithLen(len(s.static)) }
func (s *Sheet) randomIndexWithLen(n int) int {
	if UseSequentialDailyWord {
		now := time.Now().Truncate(24 * time.Hour)
		start := dailyWordStartDate.Truncate(24 * time.Hour)
		days := int(now.Sub(start).Hours() / 24)
		return days % n
	}
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(n)
}

func interfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, v := range ss {
		out[i] = v
	}
	return out
}

func parseWordData(row []interface{}) (*WordData, error) {
	expectedColumns := 13
	if len(row) < expectedColumns {
		return nil, fmt.Errorf("invalid row format: expected at least %d columns, got %d", expectedColumns, len(row))
	}

	data := &WordData{
		Answer:         fmt.Sprint(row[0]),
		Hints:          make(map[string]string),
		Categories:     make(map[string]string),
		CategoryEmojis: make(map[string]string),
	}

	categoryIndex := 0
	for i := 1; i < expectedColumns; i += 3 {
		if i+2 >= len(row) {
			fmt.Printf("Warning: Row has fewer columns than expected (%d) for category block starting at index %d.\n", len(row), i)
			break
		}

		category := fmt.Sprint(row[i])
		hint := fmt.Sprint(row[i+1])
		emoji := fmt.Sprint(row[i+2])

		if category == "" {
			continue
		}
		data.Categories[category] = fmt.Sprintf("Category %c", 'A'+categoryIndex)
		data.Hints[category] = hint
		data.CategoryEmojis[category] = emoji
		data.CategoryOrder = append(data.CategoryOrder, category)
		categoryIndex++
	}

	if len(data.Hints) == 0 {
		return nil, fmt.Errorf("no valid categories parsed from the row")
	}

	return data, nil
}

func (s *Sheet) InitAnalytics(sheetID string) {
	s.analytics = &Analytics{
		sheetID:    sheetID,
		eventsChan: make(chan Event, 1000),
		service:    s.service,
		sheetCache: make(map[string]struct{}),
	}
	go s.analytics.processEvents()
}

func (a *Analytics) processEvents() {
	for event := range a.eventsChan {
		a.writeEventToSheet(event)
	}
}

func (a *Analytics) writeEventToSheet(event Event) {

	sheetName := fmt.Sprintf("Game-%s", event.GameID)

	if _, exists := a.sheetCache[sheetName]; !exists {
		exists, err := a.sheetExists(sheetName)
		if err != nil {
			log.Printf("Sheet check failed: %v", err)
			return
		}

		if !exists {
			if err := a.createSheetWithHeaders(sheetName); err != nil {
				log.Printf("Sheet creation failed: %v", err)
				return
			}
		}

		a.sheetCache[sheetName] = struct{}{}
	}

	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{
				event.Timestamp.Format(time.RFC3339),
				event.EventType,
				event.Data["correct"],
				event.Data["guess"],
				event.Data["category"],
				event.PlayerID,
			},
		},
	}

	_, err := a.service.Spreadsheets.Values.Append(
		a.sheetID,
		fmt.Sprintf("%s!A1", sheetName),
		values,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		log.Printf("Failed to log event: %v", err)
	}
}

func (s *Sheet) LogEvent(event Event) {
	if s.analytics == nil {
		return
	}
	select {
	case s.analytics.eventsChan <- event:
	default:
	}
}

func (a *Analytics) sheetExists(sheetName string) (bool, error) {
	resp, err := a.service.Spreadsheets.Get(a.sheetID).Do()
	if err != nil {
		return false, err
	}

	for _, sheet := range resp.Sheets {
		if sheet.Properties.Title == sheetName {
			return true, nil
		}
	}
	return false, nil
}

func (a *Analytics) createSheetWithHeaders(sheetName string) error {

	addReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetName,
					},
				},
			},
		},
	}

	_, err := a.service.Spreadsheets.BatchUpdate(a.sheetID, addReq).Do()
	if err != nil {
		return err
	}

	headerValues := &sheets.ValueRange{
		Values: [][]interface{}{
			{"Timestamp", "Event Type", "Correct", "Guess", "Category", "PlayerID"},
		},
	}

	_, err = a.service.Spreadsheets.Values.Update(
		a.sheetID,
		fmt.Sprintf("%s!A1", sheetName),
		headerValues,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	statsValues := &sheets.ValueRange{
		Values: [][]interface{}{
			{"STATISTICS"},
			{"Total Players", "=COUNTA(UNIQUE(F2:F1000))"},
			{"Players Solved", "=COUNTA(UNIQUE(FILTER(F2:F, B2:B=\"guess\", C2:C=TRUE)))"},
			{"Solve Rate", "=I3/I2"},
		},
	}

	_, err = a.service.Spreadsheets.Values.Update(
		a.sheetID,
		fmt.Sprintf("%s!H1", sheetName),
		statsValues,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	rankHeaderValues := &sheets.ValueRange{
		Values: [][]interface{}{
			{"PLAYER RANKINGS"},
			{"PlayerID"},
		},
	}

	_, err = a.service.Spreadsheets.Values.Update(
		a.sheetID,
		fmt.Sprintf("%s!K1", sheetName),
		rankHeaderValues,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	playerListFormula := &sheets.ValueRange{
		Values: [][]interface{}{
			{"=UNIQUE(FILTER(F2:F, B2:B=\"guess\", C2:C=TRUE))"},
		},
	}

	_, err = a.service.Spreadsheets.Values.Update(
		a.sheetID,
		fmt.Sprintf("%s!K3", sheetName),
		playerListFormula,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	return nil
}

func (s *Sheet) GetPlayerStats(gameID, playerID string) (*PlayerStats, error) {
	sheetName := fmt.Sprintf("Game-%s", strings.Trim(gameID, "\""))

	if _, exists := s.analytics.sheetCache[sheetName]; !exists {
		exists, err := s.analytics.sheetExists(sheetName)
		if err != nil {
			log.Printf("Sheet check failed: %v", err)
			return &PlayerStats{
				TotalPlayers:  1,
				PlayersSolved: 1,
				PlayerRank:    1,
			}, nil
		}

		if !exists {
			log.Printf("Returning 1s: %v", sheetName)
			return &PlayerStats{
				TotalPlayers:  1,
				PlayersSolved: 1,
				PlayerRank:    1,
			}, nil
		}

		s.analytics.sheetCache[sheetName] = struct{}{}
	}

	statsRange := fmt.Sprintf("%s!I2:I3", sheetName)
	statsResp, err := s.service.Spreadsheets.Values.Get(s.sheetID, statsRange).Do()
	if err != nil {
		log.Printf("Warning: Failed to get stats: %v", err)
		return &PlayerStats{
			TotalPlayers:  1,
			PlayersSolved: 1,
			PlayerRank:    1,
		}, nil
	}

	stats := &PlayerStats{
		TotalPlayers:  1,
		PlayersSolved: 1,
		PlayerRank:    1,
	}

	if len(statsResp.Values) >= 1 && len(statsResp.Values[0]) >= 1 {
		totalStr := fmt.Sprint(statsResp.Values[0][0])
		totalPlayers, err := strconv.Atoi(totalStr)
		if err == nil && totalPlayers > 0 {
			stats.TotalPlayers = totalPlayers
		}
	}

	if len(statsResp.Values) >= 2 && len(statsResp.Values[1]) >= 1 {
		solvedStr := fmt.Sprint(statsResp.Values[1][0])
		playersSolved, err := strconv.Atoi(solvedStr)
		if err == nil && playersSolved > 0 {
			stats.PlayersSolved = playersSolved
		}
	}

	rankingRange := fmt.Sprintf("%s!K3:K", sheetName)
	rankingResp, err := s.service.Spreadsheets.Values.Get(s.sheetID, rankingRange).Do()
	if err != nil {
		log.Printf("Warning: Failed to get player rankings: %v", err)
		return stats, nil
	}

	for rank, row := range rankingResp.Values {
		rowPlayerID := fmt.Sprint(row[0])
		if rowPlayerID == "" {
			continue
		}

		if strings.TrimSpace(rowPlayerID) == strings.TrimSpace(playerID) {
			stats.PlayerRank = rank
			break
		}
	}
	return stats, nil
}
