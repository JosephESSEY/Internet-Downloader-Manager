package storage

import (
    "database/sql"
    "time"
    
    _ "github.com/mattn/go-sqlite3"
)

// DownloadStatus represents the status of a download
type DownloadStatus int

const (
    StatusPending DownloadStatus = iota
    StatusDownloading
    StatusPaused
    StatusCompleted
    StatusFailed
    StatusCancelled
)

func (s DownloadStatus) String() string {
    switch s {
    case StatusPending:
        return "Pending"
    case StatusDownloading:
        return "Downloading"
    case StatusPaused:
        return "Paused"
    case StatusCompleted:
        return "Completed"
    case StatusFailed:
        return "Failed"
    case StatusCancelled:
        return "Cancelled"
    default:
        return "Unknown"
    }
}

// Download represents a download item
type Download struct {
    ID          int64         `json:"id"`
    URL         string        `json:"url"`
    Filename    string        `json:"filename"`
    Path        string        `json:"path"`
    Size        int64         `json:"size"`
    Downloaded  int64         `json:"downloaded"`
    Status      DownloadStatus `json:"status"`
    Speed       int64         `json:"speed"`
    Progress    float64       `json:"progress"`
    CreatedAt   time.Time     `json:"created_at"`
    StartedAt   *time.Time    `json:"started_at,omitempty"`
    CompletedAt *time.Time    `json:"completed_at,omitempty"`
    Error       string        `json:"error,omitempty"`
    Chunks      int           `json:"chunks"`
}

func InitDB() (*sql.DB, error) {
    db, err := sql.Open("sqlite3", "idm.db")
    if err != nil {
        return nil, err
    }

    if err := createTables(db); err != nil {
        return nil, err
    }

    return db, nil
}

func createTables(db *sql.DB) error {
    query := `
    CREATE TABLE IF NOT EXISTS downloads (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT NOT NULL,
        filename TEXT NOT NULL,
        path TEXT NOT NULL,
        size INTEGER DEFAULT 0,
        downloaded INTEGER DEFAULT 0,
        status INTEGER DEFAULT 0,
        speed INTEGER DEFAULT 0,
        progress REAL DEFAULT 0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        started_at DATETIME,
        completed_at DATETIME,
        error TEXT,
        chunks INTEGER DEFAULT 1
    );`

    _, err := db.Exec(query)
    return err
}

func SaveDownload(db *sql.DB, download *Download) (int64, error) {
    query := `
    INSERT INTO downloads (url, filename, path, size, downloaded, status, chunks, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := db.Exec(query,
        download.URL,
        download.Filename,
        download.Path,
        download.Size,
        download.Downloaded,
        int(download.Status),
        download.Chunks,
        download.CreatedAt,
    )

    if err != nil {
        return 0, err
    }

    return result.LastInsertId()
}

func UpdateDownload(db *sql.DB, download *Download) error {
    query := `
    UPDATE downloads 
    SET downloaded = ?, status = ?, speed = ?, progress = ?, started_at = ?, completed_at = ?, error = ?
    WHERE id = ?`

    _, err := db.Exec(query,
        download.Downloaded,
        int(download.Status),
        download.Speed,
        download.Progress,
        download.StartedAt,
        download.CompletedAt,
        download.Error,
        download.ID,
    )

    return err
}

func GetDownload(db *sql.DB, id int64) (*Download, error) {
    query := `
    SELECT id, url, filename, path, size, downloaded, status, speed, progress, 
           created_at, started_at, completed_at, error, chunks
    FROM downloads WHERE id = ?`

    row := db.QueryRow(query, id)

    download := &Download{}
    var startedAt, completedAt sql.NullTime

    err := row.Scan(
        &download.ID,
        &download.URL,
        &download.Filename,
        &download.Path,
        &download.Size,
        &download.Downloaded,
        (*int)(&download.Status),
        &download.Speed,
        &download.Progress,
        &download.CreatedAt,
        &startedAt,
        &completedAt,
        &download.Error,
        &download.Chunks,
    )

    if err != nil {
        return nil, err
    }

    if startedAt.Valid {
        download.StartedAt = &startedAt.Time
    }
    if completedAt.Valid {
        download.CompletedAt = &completedAt.Time
    }

    return download, nil
}

func GetAllDownloads(db *sql.DB) ([]*Download, error) {
    query := `
    SELECT id, url, filename, path, size, downloaded, status, speed, progress, 
           created_at, started_at, completed_at, error, chunks
    FROM downloads ORDER BY created_at DESC`

    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var downloads []*Download

    for rows.Next() {
        download := &Download{}
        var startedAt, completedAt sql.NullTime

        err := rows.Scan(
            &download.ID,
            &download.URL,
            &download.Filename,
            &download.Path,
            &download.Size,
            &download.Downloaded,
            (*int)(&download.Status),
            &download.Speed,
            &download.Progress,
            &download.CreatedAt,
            &startedAt,
            &completedAt,
            &download.Error,
            &download.Chunks,
        )

        if err != nil {
            return nil, err
        }

        if startedAt.Valid {
            download.StartedAt = &startedAt.Time
        }
        if completedAt.Valid {
            download.CompletedAt = &completedAt.Time
        }

        downloads = append(downloads, download)
    }

    return downloads, rows.Err()
}

func DeleteDownload(db *sql.DB, id int64) error {
    query := "DELETE FROM downloads WHERE id = ?"
    _, err := db.Exec(query, id)
    return err
}