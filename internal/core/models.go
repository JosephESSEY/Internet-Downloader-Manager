package core

import (
    "time"
    "idm-go/internal/storage"
)

// Use storage types
type Download = storage.Download
type DownloadStatus = storage.DownloadStatus

// Re-export constants
const (
    StatusPending     = storage.StatusPending
    StatusDownloading = storage.StatusDownloading
    StatusPaused      = storage.StatusPaused
    StatusCompleted   = storage.StatusCompleted
    StatusFailed      = storage.StatusFailed
    StatusCancelled   = storage.StatusCancelled
)

type DownloadConfig struct {
    MaxConcurrentDownloads int
    ChunkSize              int64
    MaxSpeed               int64 // bytes per second, 0 for unlimited
    RetryAttempts          int
    UserAgent              string
    Timeout                time.Duration
}

func DefaultConfig() *DownloadConfig {
    return &DownloadConfig{
        MaxConcurrentDownloads: 3,
        ChunkSize:              1024 * 1024, // 1MB
        MaxSpeed:               0,            // unlimited
        RetryAttempts:          3,
        UserAgent:              "IDM-Go/1.0",
        Timeout:                30 * time.Second,
    }
}