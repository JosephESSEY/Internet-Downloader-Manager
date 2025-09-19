package ui

import (
    "path/filepath"
    "strings"
    "os"
    "idm-go/internal/core"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/storage"
    "fyne.io/fyne/v2/widget"
)

type AddDownloadDialog struct {
    dialog          dialog.Dialog
    urlEntry        *widget.Entry
    pathEntry       *widget.Entry
    chunksSelect    *widget.Select
    downloadManager *core.DownloadManager
    callback        func(*core.Download)
}

func NewAddDownloadDialog(parent fyne.Window, dm *core.DownloadManager, callback func(*core.Download)) *AddDownloadDialog {
    add := &AddDownloadDialog{
        downloadManager: dm,
        callback:        callback,
    }

    add.createDialog(parent)
    return add
}

func (add *AddDownloadDialog) createDialog(parent fyne.Window) {
    // URL entry
    add.urlEntry = widget.NewEntry()
    add.urlEntry.SetPlaceHolder("Enter download URL...")
    add.urlEntry.Validator = func(s string) error {
        if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "ftp://") {
            return nil // Allow empty for now, will validate on submit
        }
        return nil
    }

    // Path entry with browse button
    add.pathEntry = widget.NewEntry()
    add.pathEntry.SetText(getDefaultDownloadPath())

    browseButton := widget.NewButton("Browse", func() {
        dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
            if err == nil && folder != nil {
                add.pathEntry.SetText(folder.Path())
            }
        }, parent)
    })

    pathContainer := container.NewBorder(nil, nil, nil, browseButton, add.pathEntry)

    // Chunks selection
    add.chunksSelect = widget.NewSelect([]string{"1", "2", "4", "8", "16"}, nil)
    add.chunksSelect.SetSelected("4")

    // Buttons
    addButton := widget.NewButton("Add Download", add.addDownload)
    addButton.Importance = widget.HighImportance

    cancelButton := widget.NewButton("Cancel", func() {
        add.dialog.Hide()
    })

    buttons := container.NewHBox(cancelButton, addButton)

    // Form
    form := container.NewVBox(
        widget.NewLabel("Download URL:"),
        add.urlEntry,
        widget.NewSeparator(),
        widget.NewLabel("Download Path:"),
        pathContainer,
        widget.NewSeparator(),
        widget.NewLabel("Number of Chunks:"),
        add.chunksSelect,
        widget.NewSeparator(),
        buttons,
    )

    add.dialog = dialog.NewCustom("Add New Download", "", form, parent)
    add.dialog.Resize(fyne.NewSize(500, 300))
}

func (add *AddDownloadDialog) addDownload() {
    url := strings.TrimSpace(add.urlEntry.Text)
    path := strings.TrimSpace(add.pathEntry.Text)

    // Validation
    if url == "" {
        dialog.ShowError(fmt.Errorf("URL is required"), add.dialog.(fyne.Window))
        return
    }

    if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "ftp://") {
        dialog.ShowError(fmt.Errorf("Invalid URL format"), add.dialog.(fyne.Window))
        return
    }

    if path == "" {
        path = getDefaultDownloadPath()
    }

    // Check if path exists
    if _, err := os.Stat(path); os.IsNotExist(err) {
        if err := os.MkdirAll(path, 0755); err != nil {
            dialog.ShowError(fmt.Errorf("Cannot create download directory: %v", err), add.dialog.(fyne.Window))
            return
        }
    }

    // Add download
    download, err := add.downloadManager.AddDownload(url, path)
    if err != nil {
        dialog.ShowError(err, add.dialog.(fyne.Window))
        return
    }

    // Set chunks if specified
    if chunks := add.chunksSelect.Selected; chunks != "" {
        if chunksInt, err := strconv.Atoi(chunks); err == nil {
            download.Chunks = chunksInt
        }
    }

    add.dialog.Hide()
    
    if add.callback != nil {
        add.callback(download)
    }

    // Show success notification
    dialog.ShowInformation("Success", 
        fmt.Sprintf("Download added successfully!\nFile: %s", download.Filename), 
        add.dialog.(fyne.Window))
}

func (add *AddDownloadDialog) Show() {
    add.dialog.Show()
}

func getDefaultDownloadPath() string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "."
    }
    return filepath.Join(homeDir, "Downloads")
}