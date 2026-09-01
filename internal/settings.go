package internal

import (
	"fmt"
	"strings"
	"time"

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
	timeout := m.timeout
	m.mu.RUnlock()
	return []contracts.SettingDef{
		{
			Key:         "allowed_commands",
			Label:       "Allowed Commands",
			Type:        contracts.SettingTypeString,
			Value:       allowed,
			Default:     "",
			Description: "Comma-separated first tokens (EXECUTOR_ALLOWED_COMMANDS). Empty denies all; * requires EXECUTOR_ALLOW_WILDCARD=1.",
			Group:       "Security",
		},
		{
			Key:         "timeout",
			Label:       "Execution Timeout",
			Type:        contracts.SettingTypeString,
			Value:       timeout.String(),
			Default:     defaultTimeout.String(),
			Description: "Default max runtime when the mesh context has no deadline (EXECUTOR_TIMEOUT).",
			Group:       "Security",
		},
	}
}

func (m *Module) updateSetting(key, value string) error {
	switch key {
	case "allowed_commands", envAllowedCommands:
		value = strings.TrimSpace(value)
		if err := validateAllowedCommandsValue(value); err != nil {
			return err
		}
		m.mu.Lock()
		m.allowedCommands = value
		m.mu.Unlock()
		return nil
	case "timeout", envTimeout:
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("timeout must not be empty")
		}
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return fmt.Errorf("invalid timeout %q: must be a positive duration (e.g. 5m)", value)
		}
		m.mu.Lock()
		m.timeout = d
		m.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
}
