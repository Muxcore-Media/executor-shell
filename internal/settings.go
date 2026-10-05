package internal

import (
	"fmt"
	"strings"

	"github.com/Muxcore-Media/core/pkg/contracts"
)

func (m *Module) Settings() []contracts.SettingDef {
	return m.settingsDefs()
}

func (m *Module) UpdateSetting(key, value string) error {
	return m.updateSetting(key, value)
}

func (m *Module) settingsDefs() []contracts.SettingDef {
	m.mu.RLock()
	allowed := m.allowedCommands
	m.mu.RUnlock()
	return []contracts.SettingDef{
		{
			Key:         settingAllowedCommands,
			Label:       "Allowed Commands",
			Type:        contracts.SettingTypeString,
			Value:       allowed,
			Default:     "",
			Description: "Comma-separated command prefixes (EXECUTOR_ALLOWED_COMMANDS). Empty denies all; avoid * in production.",
			Group:       "Security",
		},
	}
}

// settingAllowedCommands is the settings key for the command allow-list.
const settingAllowedCommands = "allowed_commands"

func (m *Module) updateSetting(key, value string) error {
	switch key {
	case settingAllowedCommands, "EXECUTOR_ALLOWED_COMMANDS":
		m.mu.Lock()
		m.allowedCommands = strings.TrimSpace(value)
		m.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
}
