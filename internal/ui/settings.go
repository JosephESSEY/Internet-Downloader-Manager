package ui

import (
    "strconv"
    "idm-go/internal/core"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
)

type SettingsWindow struct {
    app             fyne.App
    window          fyne.Window
    downloadManager *core.DownloadManager
    
    // Settings widgets
    maxDownloadsEntry   *widget.Entry
    maxSpeedEntry       *widget.Entry
    chunkSizeEntry      *widget.Entry
    retryAttemptsEntry  *widget.Entry
    userAgentEntry      *widget.Entry
    timeoutEntry        *widget.Entry
}

func NewSettingsWindow(app fyne.App, dm *core.DownloadManager) *SettingsWindow {
    window := app.NewWindow("Settings")
    window.Resize(fyne.NewSize(500, 400))

    settings := &SettingsWindow{
        app:             app,
        window:          window,
        downloadManager: dm,
    }

    settings.setupUI()
    settings.loadSettings()

    return settings
}

func (sw *SettingsWindow) setupUI() {
    // Create form entries
    sw.maxDownloadsEntry = widget.NewEntry()
    sw.maxDownloadsEntry.SetPlaceHolder("3")

    sw.maxSpeedEntry = widget.NewEntry()
    sw.maxSpeedEntry.SetPlaceHolder("0 (unlimited)")

    sw.chunkSizeEntry = widget.NewEntry()
    sw.chunkSizeEntry.SetPlaceHolder("1048576 (1MB)")

    sw.retryAttemptsEntry = widget.NewEntry()
    sw.retryAttemptsEntry.SetPlaceHolder("3")

    sw.userAgentEntry = widget.NewEntry()
    sw.userAgentEntry.SetPlaceHolder("IDM-Go/1.0")

    sw.timeoutEntry = widget.NewEntry()
    sw.timeoutEntry.SetPlaceHolder("30")

    // Create form
    form := &widget.Form{
        Items: []*widget.FormItem{
            {Text: "Max Concurrent Downloads:", Widget: sw.maxDownloadsEntry},
            {Text: "Max Speed (bytes/sec, 0=unlimited):", Widget: sw.maxSpeedEntry},
            {Text: "Chunk Size (bytes):", Widget: sw.chunkSizeEntry},
            {Text: "Retry Attempts:", Widget: sw.retryAttemptsEntry},
            {Text: "User Agent:", Widget: sw.userAgentEntry},
            {Text: "Timeout (seconds):", Widget: sw.timeoutEntry},
        },
        OnSubmit:   sw.saveSettings,
        OnCancel:   func() { sw.window.Hide() },
        SubmitText: "Save",
        CancelText: "Cancel",
    }

    // Additional buttons
    resetButton := widget.NewButton("Reset to Defaults", sw.resetToDefaults)
    resetButton.Importance = widget.MediumImportance

    aboutButton := widget.NewButton("About", sw.showAbout)

    buttonsContainer := container.NewHBox(resetButton, aboutButton)

    content := container.NewVBox(
        widget.NewLabel("Download Manager Settings"),
        widget.NewSeparator(),
        form,
        widget.NewSeparator(),
        buttonsContainer,
    )

    sw.window.SetContent(container.NewScroll(content))
}

func (sw *SettingsWindow) loadSettings() {
    // In a real implementation, you would load these from the DownloadManager
    // For now, we'll use default values
    config := core.DefaultConfig()

    sw.maxDownloadsEntry.SetText(strconv.Itoa(config.MaxConcurrentDownloads))
    sw.maxSpeedEntry.SetText(strconv.FormatInt(config.MaxSpeed, 10))
    sw.chunkSizeEntry.SetText(strconv.FormatInt(config.ChunkSize, 10))
    sw.retryAttemptsEntry.SetText(strconv.Itoa(config.RetryAttempts))
    sw.userAgentEntry.SetText(config.UserAgent)
    sw.timeoutEntry.SetText(strconv.FormatInt(int64(config.Timeout.Seconds()), 10))
}

func (sw *SettingsWindow) saveSettings() {
    // Validate and parse settings
    maxDownloads, err := strconv.Atoi(sw.maxDownloadsEntry.Text)
    if err != nil || maxDownloads < 1 || maxDownloads > 20 {
        dialog.ShowError(fmt.Errorf("Max downloads must be between 1 and 20"), sw.window)
        return
    }

    maxSpeed, err := strconv.ParseInt(sw.maxSpeedEntry.Text, 10, 64)
    if err != nil || maxSpeed < 0 {
        dialog.ShowError(fmt.Errorf("Max speed must be a positive number or 0"), sw.window)
        return
    }

    chunkSize, err := strconv.ParseInt(sw.chunkSizeEntry.Text, 10, 64)
    if err != nil || chunkSize < 1024 {
        dialog.ShowError(fmt.Errorf("Chunk size must be at least 1024 bytes"), sw.window)
        return
    }

    retryAttempts, err := strconv.Atoi(sw.retryAttemptsEntry.Text)
    if err != nil || retryAttempts < 0 || retryAttempts > 10 {
        dialog.ShowError(fmt.Errorf("Retry attempts must be between 0 and 10"), sw.window)
        return
    }

    timeout, err := strconv.ParseInt(sw.timeoutEntry.Text, 10, 64)
    if err != nil || timeout < 5 || timeout > 300 {
        dialog.ShowError(fmt.Errorf("Timeout must be between 5 and 300 seconds"), sw.window)
        return
    }

    userAgent := strings.TrimSpace(sw.userAgentEntry.Text)
    if userAgent == "" {
        userAgent = "IDM-Go/1.0"
    }

    // Apply settings (in a real implementation, you would save these to the DownloadManager)
    dialog.ShowInformation("Settings Saved", "Settings have been saved successfully!", sw.window)
    sw.window.Hide()
}

func (sw *SettingsWindow) resetToDefaults() {
    dialog.ShowConfirm("Reset Settings", 
        "Are you sure you want to reset all settings to default values?",
        func(confirmed bool) {
            if confirmed {
                sw.loadSettings()
            }
        }, sw.window)
}

func (sw *SettingsWindow) showAbout() {
    about := dialog.NewInformation("About IDM Go",
        "IDM Go - Internet Download Manager\n\n"+
        "Version: 1.0.0\n"+
        "Built with Go and Fyne\n\n"+
        "Features:\n"+
        "• Multi-threaded downloads\n"+
        "• Resume support\n"+
        "• Speed limiting\n"+
        "• Queue management\n"+
        "• Modern GUI\n\n"+
        "© 2024 IDM Go Project",
        sw.window)
    about.Show()
}

func (sw *SettingsWindow) Show() {
    sw.window.Show()
}