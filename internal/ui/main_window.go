package ui

import (
    "fmt"
    "path/filepath"
    "strconv"
    "time"
    "idm-go/internal/core"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/theme"
)

type MainWindow struct {
    app             fyne.App
    window          fyne.Window
    downloadManager *core.DownloadManager
    downloadsList   *widget.List
    downloads       []*core.Download
    statusBar       *widget.Label
}

func NewMainWindow(app fyne.App, dm *core.DownloadManager) *MainWindow {
    window := app.NewWindow("IDM Go - Internet Download Manager")
    window.Resize(fyne.NewSize(800, 600))
    window.SetMaster()

    mw := &MainWindow{
        app:             app,
        window:          window,
        downloadManager: dm,
        statusBar:       widget.NewLabel("Ready"),
    }

    mw.setupUI()
    mw.loadDownloads()
    
    // Register callback for download updates
    dm.AddCallback(mw.onDownloadUpdate)

    return mw
}

func (mw *MainWindow) setupUI() {
    // Create toolbar
    toolbar := mw.createToolbar()

    // Create downloads list
    mw.createDownloadsList()

    // Create status bar
    statusContainer := container.NewBorder(nil, nil, mw.statusBar, nil)

    // Main content
    content := container.NewBorder(
        toolbar,        // top
        statusContainer, // bottom
        nil,            // left
        nil,            // right
        mw.downloadsList, // center
    )

    mw.window.SetContent(content)
    mw.updateStatusBar()
}

func (mw *MainWindow) createToolbar() *widget.Toolbar {
    toolbar := widget.NewToolbar(
        widget.NewToolbarAction(theme.ContentAddIcon(), mw.showAddDownloadDialog),
        widget.NewToolbarSeparator(),
        widget.NewToolbarAction(theme.MediaPlayIcon(), mw.startSelectedDownload),
        widget.NewToolbarAction(theme.MediaPauseIcon(), mw.pauseSelectedDownload),
        widget.NewToolbarAction(theme.MediaStopIcon(), mw.cancelSelectedDownload),
        widget.NewToolbarSeparator(),
        widget.NewToolbarAction(theme.DeleteIcon(), mw.deleteSelectedDownload),
        widget.NewToolbarAction(theme.ViewRefreshIcon(), mw.refreshDownloads),
        widget.NewToolbarSeparator(),
        widget.NewToolbarAction(theme.SettingsIcon(), mw.showSettings),
    )

    return toolbar
}

func (mw *MainWindow) createDownloadsList() {
    mw.downloadsList = widget.NewList(
        func() int {
            return len(mw.downloads)
        },
        func() fyne.CanvasObject {
            return mw.createDownloadItem()
        },
        func(id widget.ListItemID, item fyne.CanvasObject) {
            if id >= len(mw.downloads) {
                return
            }
            mw.updateDownloadItem(item, mw.downloads[id])
        },
    )
}

func (mw *MainWindow) createDownloadItem() fyne.CanvasObject {
    filename := widget.NewLabel("")
    filename.TextStyle.Bold = true
    
    url := widget.NewLabel("")
    url.Truncation = fyne.TextTruncateEllipsis
    
    status := widget.NewLabel("")
    
    progress := widget.NewProgressBar()
    
    size := widget.NewLabel("")
    speed := widget.NewLabel("")
    
    infoContainer := container.NewHBox(size, speed, status)
    
    return container.NewVBox(
        filename,
        url,
        progress,
        infoContainer,
    )
}

func (mw *MainWindow) updateDownloadItem(item fyne.CanvasObject, download *core.Download) {
    container := item.(*container.VBox)
    
    filename := container.Objects[0].(*widget.Label)
    url := container.Objects[1].(*widget.Label)
    progress := container.Objects[2].(*widget.ProgressBar)
    infoContainer := container.Objects[3].(*container.HBox)
    
    size := infoContainer.Objects[0].(*widget.Label)
    speed := infoContainer.Objects[1].(*widget.Label)
    status := infoContainer.Objects[2].(*widget.Label)
    
    filename.SetText(download.Filename)
    url.SetText(download.URL)
    status.SetText(download.Status.String())
    
    // Update progress
    progress.SetValue(download.Progress / 100.0)
    
    // Format size
    if download.Size > 0 {
        totalSize := formatBytes(download.Size)
        downloadedSize := formatBytes(download.Downloaded)
        size.SetText(fmt.Sprintf("%s / %s", downloadedSize, totalSize))
    } else {
        size.SetText(formatBytes(download.Downloaded))
    }
    
    // Format speed
    if download.Status == core.StatusDownloading && download.Speed > 0 {
        speed.SetText(fmt.Sprintf("%s/s", formatBytes(download.Speed)))
    } else {
        speed.SetText("")
    }
    
    // Set status color
    switch download.Status {
    case core.StatusCompleted:
        status.Importance = widget.SuccessImportance
    case core.StatusFailed:
        status.Importance = widget.DangerImportance
    case core.StatusDownloading:
        status.Importance = widget.MediumImportance
    default:
        status.Importance = widget.MediumImportance
    }
}

func (mw *MainWindow) showAddDownloadDialog() {
    dialog := NewAddDownloadDialog(mw.window, mw.downloadManager, mw.onDownloadAdded)
    dialog.Show()
}

func (mw *MainWindow) onDownloadAdded(download *core.Download) {
    mw.loadDownloads()
    mw.updateStatusBar()
}

func (mw *MainWindow) startSelectedDownload() {
    selected := mw.downloadsList.GetSelected()
    if selected >= 0 && selected < len(mw.downloads) {
        download := mw.downloads[selected]
        if err := mw.downloadManager.StartDownload(download.ID); err != nil {
            dialog.ShowError(err, mw.window)
        }
    }
}

func (mw *MainWindow) pauseSelectedDownload() {
    selected := mw.downloadsList.GetSelected()
    if selected >= 0 && selected < len(mw.downloads) {
        download := mw.downloads[selected]
        if err := mw.downloadManager.PauseDownload(download.ID); err != nil {
            dialog.ShowError(err, mw.window)
        }
    }
}

func (mw *MainWindow) cancelSelectedDownload() {
    selected := mw.downloadsList.GetSelected()
    if selected >= 0 && selected < len(mw.downloads) {
        download := mw.downloads[selected]
        
        dialog.ShowConfirm("Cancel Download", 
            "Are you sure you want to cancel this download?",
            func(confirmed bool) {
                if confirmed {
                    if err := mw.downloadManager.CancelDownload(download.ID); err != nil {
                        dialog.ShowError(err, mw.window)
                    }
                }
            }, mw.window)
    }
}

func (mw *MainWindow) deleteSelectedDownload() {
    selected := mw.downloadsList.GetSelected()
    if selected >= 0 && selected < len(mw.downloads) {
        download := mw.downloads[selected]
        
        dialog.ShowConfirm("Delete Download", 
            "Are you sure you want to delete this download from the list?",
            func(confirmed bool) {
                if confirmed {
                    // Cancel if running
                    if download.Status == core.StatusDownloading {
                        mw.downloadManager.CancelDownload(download.ID)
                    }
                    mw.loadDownloads()
                }
            }, mw.window)
    }
}

func (mw *MainWindow) refreshDownloads() {
    mw.loadDownloads()
    mw.updateStatusBar()
}

func (mw *MainWindow) showSettings() {
    settings := NewSettingsWindow(mw.app, mw.downloadManager)
    settings.Show()
}

func (mw *MainWindow) loadDownloads() {
    downloads, err := mw.downloadManager.GetDownloads()
    if err != nil {
        dialog.ShowError(err, mw.window)
        return
    }
    
    mw.downloads = downloads
    mw.downloadsList.Refresh()
}

func (mw *MainWindow) onDownloadUpdate(download *core.Download) {
    // Find and update the download in our list
    for i, d := range mw.downloads {
        if d.ID == download.ID {
            mw.downloads[i] = download
            break
        }
    }
    
    // Refresh the list on the main thread
    mw.downloadsList.Refresh()
    mw.updateStatusBar()
}

func (mw *MainWindow) updateStatusBar() {
    downloading := 0
    completed := 0
    failed := 0
    
    for _, download := range mw.downloads {
        switch download.Status {
        case core.StatusDownloading:
            downloading++
        case core.StatusCompleted:
            completed++
        case core.StatusFailed:
            failed++
        }
    }
    
    status := fmt.Sprintf("Downloads: %d | Active: %d | Completed: %d | Failed: %d", 
        len(mw.downloads), downloading, completed, failed)
    mw.statusBar.SetText(status)
}

func (mw *MainWindow) ShowAndRun() {
    mw.window.ShowAndRun()
}

func formatBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}