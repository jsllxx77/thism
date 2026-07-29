package models

import "encoding/json"

type ThemePluginSettings struct {
	Version string                     `json:"version"`
	Values  map[string]json.RawMessage `json:"values"`
}

type ThemeSettings struct {
	Theme          string                         `json:"theme"`
	CustomThemes   []json.RawMessage              `json:"custom_themes"`
	PluginSettings map[string]ThemePluginSettings `json:"plugin_settings"`
}

type ThemeSettingsView struct {
	Theme          string                         `json:"theme"`
	CustomThemes   []json.RawMessage              `json:"custom_themes"`
	PluginSettings map[string]ThemePluginSettings `json:"plugin_settings"`
	Configured     bool                           `json:"configured"`
}
