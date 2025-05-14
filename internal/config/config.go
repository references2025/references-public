package config

import "os"

type Mode string

const (
	ModeProd  Mode = "prod"
	ModeLocal Mode = "local"
)

type Config struct {
	Mode                Mode
	Port                string
	BaseGameURL         string
	CredentialsJSONPath string
	WordSheetID         string
	AnalyticsSheetID    string
}

func Load() Config {
	mode := Mode(os.Getenv("MODE"))
	if mode != ModeProd {
		mode = ModeLocal
	}
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}

	return Config{
		Mode:                mode,
		Port:                get("PORT", "8080"),
		BaseGameURL:         get("BASE_GAME_URL", "http://localhost:8080"),
		CredentialsJSONPath: get("GOOGLE_CREDS_JSON", ""),
		WordSheetID:         get("WORD_SHEET_ID", ""),
		AnalyticsSheetID:    get("ANALYTICS_SHEET_ID", ""),
	}
}
