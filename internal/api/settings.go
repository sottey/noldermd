package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

const settingsFileName = "settings.json"

type Settings struct {
	Version                 int    `json:"version"`
	DarkMode                bool   `json:"darkMode"`
	DefaultView             string `json:"defaultView"`
	AutosaveEnabled         bool   `json:"autosaveEnabled"`
	AutosaveIntervalSeconds int    `json:"autosaveIntervalSeconds"`
}

type SettingsResponse struct {
	Settings Settings `json:"settings"`
	Notice   string   `json:"notice,omitempty"`
}

type SettingsPayload struct {
	DarkMode                bool   `json:"darkMode"`
	DefaultView             string `json:"defaultView"`
	AutosaveEnabled         bool   `json:"autosaveEnabled"`
	AutosaveIntervalSeconds int    `json:"autosaveIntervalSeconds"`
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, notice, err := s.loadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load settings")
		return
	}

	resp := SettingsResponse{Settings: settings}
	if notice != "" {
		resp.Notice = notice
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[SettingsPayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateSettingsPayload(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	settings, _, err := s.loadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load settings")
		return
	}

	settings.DarkMode = payload.DarkMode
	settings.DefaultView = payload.DefaultView
	settings.AutosaveEnabled = payload.AutosaveEnabled
	settings.AutosaveIntervalSeconds = payload.AutosaveIntervalSeconds
	if err := s.saveSettings(settings); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to save settings")
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) settingsFilePath() string {
	return filepath.Join(s.notesDir, settingsFileName)
}

func (s *Server) loadSettings() (Settings, string, error) {
	path := s.settingsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			settings := Settings{
				Version:                 1,
				DarkMode:                false,
				DefaultView:             "split",
				AutosaveEnabled:         false,
				AutosaveIntervalSeconds: 30,
			}
			if err := os.MkdirAll(s.notesDir, 0o755); err != nil {
				return settings, "", err
			}
			if err := s.saveSettings(settings); err != nil {
				return settings, "", err
			}
			return settings, "Created settings.json", nil
		}
		return Settings{}, "", err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, "", err
	}
	if settings.Version == 0 {
		settings.Version = 1
	}
	if settings.DefaultView == "" {
		settings.DefaultView = "split"
	}
	if settings.AutosaveIntervalSeconds == 0 {
		settings.AutosaveIntervalSeconds = 30
	}

	return settings, "", nil
}

func (s *Server) saveSettings(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.settingsFilePath(), data, 0o644)
}

func validateSettingsPayload(payload SettingsPayload) error {
	switch payload.DefaultView {
	case "edit", "preview", "split":
		// ok
	default:
		return errors.New("defaultView must be edit, preview, or split")
	}
	if payload.AutosaveIntervalSeconds < 5 {
		return errors.New("autosaveIntervalSeconds must be at least 5 seconds")
	}
	return nil
}
