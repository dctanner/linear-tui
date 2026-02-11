package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/roeyazroel/linear-tui/internal/config"
	"github.com/roeyazroel/linear-tui/internal/logger"
)

const (
	settingsModalWidth           = 110
	settingsModalScreenMargin    = 4
	settingsModalHeaderFooterRow = 2
	settingsFormItemPadding      = 1
	settingsFormBorderPadding    = 1
)

// SettingsModal manages the settings form overlay.
type SettingsModal struct {
	app                 *App
	modal               *tview.Flex
	modalBody           *tview.Flex
	modalContent        *tview.Flex
	form                *tview.Form
	endpointField       *tview.InputField
	timeoutField        *tview.InputField
	pageSizeField       *tview.InputField
	cacheTTLField       *tview.InputField
	logFileField        *tview.InputField
	logLevelField       *tview.DropDown
	logLevelOptions     []string
	themeField          *tview.DropDown
	themeOptions        []string
	themeValues         []string
	densityField        *tview.DropDown
	densityOptions      []string
	densityValues       []string
	agentWorkspaceField *tview.InputField
}

// NewSettingsModal creates a new settings modal.
func NewSettingsModal(app *App) *SettingsModal {
	sm := &SettingsModal{
		app:             app,
		logLevelOptions: []string{"debug", "info", "warning", "error"},
		themeOptions:    []string{"Linear", "High contrast", "Color-blind friendly"},
		themeValues:     []string{config.ThemeLinear, config.ThemeHighContrast, config.ThemeColorBlind},
		densityOptions:  []string{"Comfortable", "Compact"},
		densityValues:   []string{config.DensityComfortable, config.DensityCompact},
	}

	sm.form = tview.NewForm()
	sm.form.SetItemPadding(settingsFormItemPadding)
	sm.form.SetBorderPadding(settingsFormBorderPadding, settingsFormBorderPadding, settingsFormBorderPadding, settingsFormBorderPadding)
	sm.form.SetBackgroundColor(app.theme.HeaderBg)
	sm.form.SetFieldBackgroundColor(app.theme.InputBg)
	sm.form.SetFieldTextColor(app.theme.Foreground)
	sm.form.SetButtonBackgroundColor(app.theme.Accent)
	sm.form.SetButtonTextColor(app.theme.SelectionText)
	sm.form.SetLabelColor(app.theme.Foreground)
	sm.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			sm.Hide()
			return nil
		}
		return event
	})

	sm.endpointField = tview.NewInputField().
		SetLabel("API endpoint").
		SetFieldWidth(60)
	sm.form.AddFormItem(sm.endpointField)

	sm.timeoutField = tview.NewInputField().
		SetLabel("Timeout").
		SetFieldWidth(20)
	sm.form.AddFormItem(sm.timeoutField)

	sm.pageSizeField = tview.NewInputField().
		SetLabel("Page size").
		SetFieldWidth(10)
	sm.form.AddFormItem(sm.pageSizeField)

	sm.cacheTTLField = tview.NewInputField().
		SetLabel("Cache TTL").
		SetFieldWidth(20)
	sm.form.AddFormItem(sm.cacheTTLField)

	sm.logFileField = tview.NewInputField().
		SetLabel("Log file").
		SetFieldWidth(60)
	sm.form.AddFormItem(sm.logFileField)

	sm.logLevelField = tview.NewDropDown().
		SetLabel("Log level").
		SetOptions(sm.logLevelOptions, nil)
	sm.logLevelField.SetFieldWidth(20)
	sm.logLevelField.SetListStyles(
		tcell.StyleDefault.Background(app.theme.HeaderBg).Foreground(app.theme.Foreground),
		tcell.StyleDefault.Background(app.theme.Accent).Foreground(app.theme.SelectionText),
	)
	sm.form.AddFormItem(sm.logLevelField)

	sm.themeField = tview.NewDropDown().
		SetLabel("Theme").
		SetOptions(sm.themeOptions, nil)
	sm.themeField.SetFieldWidth(30)
	sm.themeField.SetListStyles(
		tcell.StyleDefault.Background(app.theme.HeaderBg).Foreground(app.theme.Foreground),
		tcell.StyleDefault.Background(app.theme.Accent).Foreground(app.theme.SelectionText),
	)
	sm.form.AddFormItem(sm.themeField)

	sm.densityField = tview.NewDropDown().
		SetLabel("Density").
		SetOptions(sm.densityOptions, nil)
	sm.densityField.SetFieldWidth(20)
	sm.densityField.SetListStyles(
		tcell.StyleDefault.Background(app.theme.HeaderBg).Foreground(app.theme.Foreground),
		tcell.StyleDefault.Background(app.theme.Accent).Foreground(app.theme.SelectionText),
	)
	sm.form.AddFormItem(sm.densityField)

	sm.agentWorkspaceField = tview.NewInputField().
		SetLabel("Agent workspace (optional; blank uses CWD)").
		SetFieldWidth(60)
	sm.form.AddFormItem(sm.agentWorkspaceField)

	sm.form.AddButton("Save", func() {
		sm.saveSettings()
	})
	sm.form.AddButton("Cancel", func() {
		sm.Hide()
	})

	titleView := tview.NewTextView()
	titleView.SetText("Settings")
	titleView.SetTextColor(app.theme.Accent)
	titleView.SetBackgroundColor(app.theme.HeaderBg)

	helpView := tview.NewTextView()
	helpView.SetText("Tab: next field | Enter: open dropdown | Esc: cancel | Agent commands: edit config.json")
	helpView.SetTextColor(app.theme.SecondaryText)
	helpView.SetBackgroundColor(app.theme.HeaderBg)
	helpView.SetTextAlign(tview.AlignCenter)

	sm.modalContent = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(titleView, 1, 0, false).
		AddItem(sm.form, 0, 1, true).
		AddItem(helpView, 1, 0, false)
	sm.modalContent.Box = tview.NewBox().SetBackgroundColor(app.theme.HeaderBg)
	sm.modalContent.SetBackgroundColor(app.theme.HeaderBg).
		SetBorder(true).
		SetBorderColor(app.theme.Accent).
		SetTitle(" Settings ").
		SetTitleColor(app.theme.Foreground)
	padding := app.density.ModalPadding
	sm.modalContent.SetBorderPadding(padding.Top, padding.Bottom, padding.Left, padding.Right)

	sm.modalBody = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(sm.modalContent, sm.settingsModalHeight(), 0, true).
		AddItem(nil, 0, 1, false)

	sm.modal = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(sm.modalBody, settingsModalWidth, 0, true).
		AddItem(nil, 0, 1, false)
	sm.modal.SetBackgroundColor(app.theme.Background)

	return sm
}

// Show displays the settings modal with current configuration values.
func (sm *SettingsModal) Show() {
	logger.Debug("tui.settings: showing settings modal")
	settings := config.SettingsFromConfig(sm.app.config)

	sm.endpointField.SetText(settings.APIEndpoint)
	sm.timeoutField.SetText(settings.Timeout)
	sm.pageSizeField.SetText(strconv.Itoa(settings.PageSize))
	sm.cacheTTLField.SetText(settings.CacheTTL)
	sm.logFileField.SetText(settings.LogFile)
	sm.setLogLevelSelection(settings.LogLevel)
	sm.setThemeSelection(settings.Theme)
	sm.setDensitySelection(settings.Density)
	sm.agentWorkspaceField.SetText(settings.AgentWorkspace)

	sm.updateModalHeight()
	sm.app.pages.AddPage("settings", sm.modal, true, true)
	sm.app.pages.SendToFront("settings")
	sm.app.app.SetFocus(sm.form)
}

// updateModalHeight recalculates and applies the modal height to fit content.
func (sm *SettingsModal) updateModalHeight() {
	if sm.modalBody == nil || sm.modalContent == nil {
		return
	}
	sm.modalBody.ResizeItem(sm.modalContent, sm.settingsModalHeight(), 0)
}

// settingsModalHeight calculates the modal height with screen-aware clamping.
func (sm *SettingsModal) settingsModalHeight() int {
	contentHeight := sm.settingsFormHeight() + settingsModalHeaderFooterRow
	padding := sm.app.density.ModalPadding
	totalHeight := contentHeight + padding.Top + padding.Bottom + 2
	maxHeight := 0
	if sm.app != nil && sm.app.pages != nil {
		_, _, _, screenHeight := sm.app.pages.GetRect()
		if screenHeight > 0 {
			maxHeight = screenHeight - settingsModalScreenMargin
			if maxHeight < 1 {
				maxHeight = screenHeight
			}
		}
	}
	if maxHeight > 0 && totalHeight > maxHeight {
		return maxHeight
	}
	return totalHeight
}

// settingsFormHeight computes the form height including padding and buttons.
func (sm *SettingsModal) settingsFormHeight() int {
	if sm.form == nil {
		return 0
	}
	itemCount := sm.form.GetFormItemCount()
	height := settingsFormBorderPadding * 2
	for i := 0; i < itemCount; i++ {
		item := sm.form.GetFormItem(i)
		itemHeight := item.GetFieldHeight()
		if itemHeight <= 0 {
			itemHeight = tview.DefaultFormFieldHeight
		}
		height += itemHeight + settingsFormItemPadding
	}
	if sm.form.GetButtonCount() > 0 {
		height++
	}
	return height
}

// Hide hides the settings modal.
func (sm *SettingsModal) Hide() {
	logger.Debug("tui.settings: hiding settings modal")
	sm.app.pages.RemovePage("settings")
	sm.app.updateFocus()
}

// HandleKey handles keyboard input for the settings modal.
func (sm *SettingsModal) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		sm.Hide()
		return nil
	}
	return event
}

// saveSettings validates input, persists settings, and applies them to the app.
func (sm *SettingsModal) saveSettings() {
	pageSizeText := strings.TrimSpace(sm.pageSizeField.GetText())
	pageSize, err := strconv.Atoi(pageSizeText)
	if err != nil {
		logger.ErrorWithErr(err, "tui.settings: invalid page size value=%s", pageSizeText)
		sm.app.updateStatusBarWithError(fmt.Errorf("page size must be a number: %w", err))
		return
	}

	_, logLevel := sm.logLevelField.GetCurrentOption()
	if logLevel == "" {
		logLevel = config.DefaultLogLevel
	}

	theme := sm.currentThemeValue()
	if theme == "" {
		theme = config.DefaultTheme
	}

	density := sm.currentDensityValue()
	if density == "" {
		density = config.DefaultDensity
	}

	settings := config.Settings{
		APIEndpoint:    strings.TrimSpace(sm.endpointField.GetText()),
		Timeout:        strings.TrimSpace(sm.timeoutField.GetText()),
		PageSize:       pageSize,
		CacheTTL:       strings.TrimSpace(sm.cacheTTLField.GetText()),
		LogFile:        strings.TrimSpace(sm.logFileField.GetText()),
		LogLevel:       logLevel,
		Theme:          theme,
		Density:        density,
		AgentCommands:  sm.app.config.AgentCommands,
		AgentWorkspace: strings.TrimSpace(sm.agentWorkspaceField.GetText()),
	}

	newCfg, err := config.ConfigFromSettings(sm.app.config.LinearAPIKey, settings)
	if err != nil {
		logger.ErrorWithErr(err, "tui.settings: failed to parse settings")
		sm.app.updateStatusBarWithError(err)
		return
	}

	settingsPath, err := config.ConfigFilePath()
	if err != nil {
		logger.ErrorWithErr(err, "tui.settings: failed to get config file path")
		sm.app.updateStatusBarWithError(err)
		return
	}

	if err := config.SaveSettings(settingsPath, settings); err != nil {
		logger.ErrorWithErr(err, "tui.settings: failed to save settings path=%s", settingsPath)
		sm.app.updateStatusBarWithError(err)
		return
	}

	logger.Debug("tui.settings: settings saved successfully path=%s", settingsPath)
	sm.Hide()
	sm.app.applySettings(newCfg)
}

// setLogLevelSelection updates the dropdown selection to match the provided level.
func (sm *SettingsModal) setLogLevelSelection(level string) {
	selected := 0
	for i, option := range sm.logLevelOptions {
		if option == config.DefaultLogLevel {
			selected = i
		}
		if option == level {
			selected = i
			break
		}
	}
	sm.logLevelField.SetCurrentOption(selected)
}

// currentThemeValue returns the currently selected theme value.
func (sm *SettingsModal) currentThemeValue() string {
	index, _ := sm.themeField.GetCurrentOption()
	if index >= 0 && index < len(sm.themeValues) {
		return sm.themeValues[index]
	}
	return ""
}

// setThemeSelection updates the dropdown selection to match the provided theme.
func (sm *SettingsModal) setThemeSelection(theme string) {
	selected := 0
	for i, value := range sm.themeValues {
		if value == config.DefaultTheme {
			selected = i
		}
		if value == theme {
			selected = i
			break
		}
	}
	sm.themeField.SetCurrentOption(selected)
}

// currentDensityValue returns the currently selected density value.
func (sm *SettingsModal) currentDensityValue() string {
	index, _ := sm.densityField.GetCurrentOption()
	if index >= 0 && index < len(sm.densityValues) {
		return sm.densityValues[index]
	}
	return ""
}

// setDensitySelection updates the dropdown selection to match the provided density.
func (sm *SettingsModal) setDensitySelection(density string) {
	selected := 0
	for i, value := range sm.densityValues {
		if value == config.DefaultDensity {
			selected = i
		}
		if value == density {
			selected = i
			break
		}
	}
	sm.densityField.SetCurrentOption(selected)
}
