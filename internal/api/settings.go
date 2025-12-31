package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const settingsFileName = "settings.json"

type Settings struct {
	Version                 int    `json:"version"`
	DarkMode                bool   `json:"darkMode"`
	DefaultView             string `json:"defaultView"`
	AutosaveEnabled         bool   `json:"autosaveEnabled"`
	AutosaveIntervalSeconds int    `json:"autosaveIntervalSeconds"`
	SidebarWidth            int    `json:"sidebarWidth"`
	DefaultFolder           string `json:"defaultFolder"`
	DailyFolder             string `json:"dailyFolder"`
	ShowTemplates           bool   `json:"showTemplates"`
}

type SettingsResponse struct {
	Settings Settings `json:"settings"`
	Notice   string   `json:"notice,omitempty"`
}

type SettingsPayload struct {
	DarkMode                *bool   `json:"darkMode,omitempty"`
	DefaultView             *string `json:"defaultView,omitempty"`
	AutosaveEnabled         *bool   `json:"autosaveEnabled,omitempty"`
	AutosaveIntervalSeconds *int    `json:"autosaveIntervalSeconds,omitempty"`
	SidebarWidth            *int    `json:"sidebarWidth,omitempty"`
	DefaultFolder           *string `json:"defaultFolder,omitempty"`
	DailyFolder             *string `json:"dailyFolder,omitempty"`
	ShowTemplates           *bool   `json:"showTemplates,omitempty"`
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

	changed := make([]string, 0, 8)
	if payload.DarkMode != nil {
		settings.DarkMode = *payload.DarkMode
		changed = append(changed, "darkMode")
	}
	if payload.DefaultView != nil {
		settings.DefaultView = *payload.DefaultView
		changed = append(changed, "defaultView")
	}
	if payload.AutosaveEnabled != nil {
		settings.AutosaveEnabled = *payload.AutosaveEnabled
		changed = append(changed, "autosaveEnabled")
	}
	if payload.AutosaveIntervalSeconds != nil {
		settings.AutosaveIntervalSeconds = *payload.AutosaveIntervalSeconds
		changed = append(changed, "autosaveIntervalSeconds")
	}
	if payload.SidebarWidth != nil {
		settings.SidebarWidth = *payload.SidebarWidth
		changed = append(changed, "sidebarWidth")
	}
	if payload.DefaultFolder != nil {
		settings.DefaultFolder = *payload.DefaultFolder
		changed = append(changed, "defaultFolder")
	}
	if payload.DailyFolder != nil {
		settings.DailyFolder = *payload.DailyFolder
		changed = append(changed, "dailyFolder")
	}
	if payload.ShowTemplates != nil {
		settings.ShowTemplates = *payload.ShowTemplates
		changed = append(changed, "showTemplates")
	}
	if err := s.saveSettings(settings); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to save settings")
		return
	}

	s.logger.Info("settings updated", "fields", strings.Join(changed, ","))
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
				Version:                 2,
				DarkMode:                false,
				DefaultView:             "split",
				AutosaveEnabled:         false,
				AutosaveIntervalSeconds: 30,
				SidebarWidth:            300,
				DefaultFolder:           "",
				DailyFolder:             "",
				ShowTemplates:           true,
			}
			if err := os.MkdirAll(s.notesDir, 0o755); err != nil {
				return settings, "", err
			}
			if err := s.saveSettings(settings); err != nil {
				return settings, "", err
			}
			s.logger.Info("settings created", "path", path)
			return settings, "Created settings.json", nil
		}
		return Settings{}, "", err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, "", err
	}
	if settings.Version == 0 {
		settings.Version = 2
	}
	if settings.DefaultView == "" {
		settings.DefaultView = "split"
	}
	if settings.AutosaveIntervalSeconds == 0 {
		settings.AutosaveIntervalSeconds = 30
	}
	if settings.SidebarWidth == 0 {
		settings.SidebarWidth = 300
	}
	if settings.DefaultFolder == "." {
		settings.DefaultFolder = ""
	}
	if settings.DailyFolder == "." {
		settings.DailyFolder = ""
	}
	if settings.Version < 2 {
		settings.ShowTemplates = true
		settings.Version = 2
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
	if payload.DefaultView != nil {
		switch *payload.DefaultView {
		case "edit", "preview", "split":
			// ok
		default:
			return errors.New("defaultView must be edit, preview, or split")
		}
	}
	if payload.AutosaveIntervalSeconds != nil && *payload.AutosaveIntervalSeconds < 5 {
		return errors.New("autosaveIntervalSeconds must be at least 5 seconds")
	}
	if payload.SidebarWidth != nil {
		if *payload.SidebarWidth < 220 || *payload.SidebarWidth > 600 {
			return errors.New("sidebarWidth must be between 220 and 600")
		}
	}
	if payload.DefaultFolder != nil {
		cleaned, err := cleanRelPath(*payload.DefaultFolder)
		if err != nil {
			return err
		}
		*payload.DefaultFolder = cleaned
	}
	if payload.DailyFolder != nil {
		cleaned, err := cleanRelPath(*payload.DailyFolder)
		if err != nil {
			return err
		}
		*payload.DailyFolder = cleaned
	}
	return nil
}
