package models

import "encoding/json"

type ThemeSettings struct {
	Theme        string            `json:"theme"`
	CustomThemes []json.RawMessage `json:"custom_themes"`
}

type ThemeSettingsView struct {
	Theme        string            `json:"theme"`
	CustomThemes []json.RawMessage `json:"custom_themes"`
	Configured   bool              `json:"configured"`
}
