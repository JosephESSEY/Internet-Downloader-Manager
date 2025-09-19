package main

import (
    "idm-go/internal/core"
    "idm-go/internal/storage"
    "idm-go/internal/ui"
    "log"

    "fyne.io/fyne/v2/app"
)

func main() {
    myApp := app.NewWithID("com.example.idm")
    
    // Initialize database
    db, err := storage.InitDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer db.Close()

    // Initialize download manager
    downloadManager := core.NewDownloadManager(db)
    
    // Create main window
    mainWindow := ui.NewMainWindow(myApp, downloadManager)
    mainWindow.ShowAndRun()
}