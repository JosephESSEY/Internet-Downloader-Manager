package core

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "sync"
    "sync/atomic"
    "time"
    "database/sql"
    "idm-go/internal/storage"
)

type DownloadManager struct {
    db              *sql.DB
    config          *DownloadConfig
    downloads       map[int64]*DownloadJob
    queue           *Queue
    activeDownloads int32
    mutex           sync.RWMutex
    callbacks       []func(*storage.Download)
}

type DownloadJob struct {
    download   *storage.Download
    ctx        context.Context
    cancel     context.CancelFunc
    client     *http.Client
    chunks     []*ChunkDownloader
    mutex      sync.RWMutex
    lastUpdate time.Time
}

type ChunkDownloader struct {
    start      int64
    end        int64
    downloaded int64
    file       *os.File
    mutex      sync.RWMutex
}

func NewDownloadManager(db *sql.DB) *DownloadManager {
    dm := &DownloadManager{
        db:        db,
        config:    DefaultConfig(),
        downloads: make(map[int64]*DownloadJob),
        queue:     NewQueue(),
    }
    
    go dm.processQueue()
    go dm.updateStats()
    
    return dm
}

func (dm *DownloadManager) AddDownload(url, path string) (*storage.Download, error) {
    // Get file info
    resp, err := http.Head(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    filename := filepath.Base(url)
    if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
        // Parse filename from Content-Disposition header
        // Simplified implementation
    }

    size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

    download := &storage.Download{
        URL:       url,
        Filename:  filename,
        Path:      path,
        Size:      size,
        Status:    storage.StatusPending,
        CreatedAt: time.Now(),
        Chunks:    4, // Default chunks
    }

    // Save to database
    id, err := storage.SaveDownload(dm.db, download)
    if err != nil {
        return nil, err
    }
    download.ID = id

    // Add to queue
    dm.queue.Add(download)
    dm.notifyCallbacks(download)

    return download, nil
}

func (dm *DownloadManager) StartDownload(id int64) error {
    download, err := storage.GetDownload(dm.db, id)
    if err != nil {
        return err
    }

    if download.Status == storage.StatusDownloading {
        return fmt.Errorf("download already in progress")
    }

    ctx, cancel := context.WithCancel(context.Background())
    
    job := &DownloadJob{
        download: download,
        ctx:      ctx,
        cancel:   cancel,
        client: &http.Client{
            Timeout: dm.config.Timeout,
        },
        lastUpdate: time.Now(),
    }

    dm.mutex.Lock()
    dm.downloads[id] = job
    dm.mutex.Unlock()

    go dm.executeDownload(job)
    
    return nil
}

// ... resto des méthodes avec storage.Download au lieu de core.Download ...

func (dm *DownloadManager) GetDownloads() ([]*storage.Download, error) {
    return storage.GetAllDownloads(dm.db)
}

func (dm *DownloadManager) AddCallback(callback func(*storage.Download)) {
    dm.callbacks = append(dm.callbacks, callback)
}

func (dm *DownloadManager) notifyCallbacks(download *storage.Download) {
    for _, callback := range dm.callbacks {
        go callback(download)
    }
}

// Reste des méthodes similaires en remplaçant core.Download par storage.Download...

func (dm *DownloadManager) executeDownload(job *DownloadJob) {
    atomic.AddInt32(&dm.activeDownloads, 1)
    defer atomic.AddInt32(&dm.activeDownloads, -1)

    download := job.download
    download.Status = StatusDownloading
    now := time.Now()
    download.StartedAt = &now
    
    dm.updateDownload(download)

    // Check if server supports range requests
    supportsRange := dm.supportsRangeRequests(download.URL)
    
    var err error
    if supportsRange && download.Size > dm.config.ChunkSize {
        err = dm.downloadWithChunks(job)
    } else {
        err = dm.downloadSingleFile(job)
    }

    if err != nil {
        download.Status = StatusFailed
        download.Error = err.Error()
    } else {
        download.Status = StatusCompleted
        now := time.Now()
        download.CompletedAt = &now
        download.Progress = 100.0
    }

    dm.updateDownload(download)
    dm.notifyCallbacks(download)

    dm.mutex.Lock()
    delete(dm.downloads, download.ID)
    dm.mutex.Unlock()
}

func (dm *DownloadManager) supportsRangeRequests(url string) bool {
    req, _ := http.NewRequest("HEAD", url, nil)
    resp, err := dm.downloads[0].client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    
    return resp.Header.Get("Accept-Ranges") == "bytes"
}

func (dm *DownloadManager) downloadWithChunks(job *DownloadJob) error {
    download := job.download
    chunkSize := download.Size / int64(download.Chunks)
    
    // Create file
    fullPath := filepath.Join(download.Path, download.Filename)
    file, err := os.Create(fullPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Pre-allocate file
    if err := file.Truncate(download.Size); err != nil {
        return err
    }

    var wg sync.WaitGroup
    errChan := make(chan error, download.Chunks)

    // Create chunks
    for i := 0; i < download.Chunks; i++ {
        start := int64(i) * chunkSize
        end := start + chunkSize - 1
        if i == download.Chunks-1 {
            end = download.Size - 1
        }

        chunk := &ChunkDownloader{
            start: start,
            end:   end,
            file:  file,
        }

        job.chunks = append(job.chunks, chunk)

        wg.Add(1)
        go func(chunk *ChunkDownloader) {
            defer wg.Done()
            if err := dm.downloadChunk(job, chunk); err != nil {
                errChan <- err
            }
        }(chunk)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}

func (dm *DownloadManager) downloadChunk(job *DownloadJob, chunk *ChunkDownloader) error {
    req, err := http.NewRequestWithContext(job.ctx, "GET", job.download.URL, nil)
    if err != nil {
        return err
    }

    req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", chunk.start, chunk.end))
    req.Header.Set("User-Agent", dm.config.UserAgent)

    resp, err := job.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Create a rate-limited reader if speed limit is set
    var reader io.Reader = resp.Body
    if dm.config.MaxSpeed > 0 {
        reader = newRateLimitedReader(resp.Body, dm.config.MaxSpeed/int64(job.download.Chunks))
    }

    buffer := make([]byte, 32*1024) // 32KB buffer
    
    for {
        select {
        case <-job.ctx.Done():
            return job.ctx.Err()
        default:
        }

        n, err := reader.Read(buffer)
        if n > 0 {
            // Write to file at specific offset
            chunk.mutex.Lock()
            _, writeErr := chunk.file.WriteAt(buffer[:n], chunk.start+chunk.downloaded)
            chunk.downloaded += int64(n)
            chunk.mutex.Unlock()

            if writeErr != nil {
                return writeErr
            }

            // Update total downloaded
            atomic.AddInt64(&job.download.Downloaded, int64(n))
        }

        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    return nil
}

func (dm *DownloadManager) downloadSingleFile(job *DownloadJob) error {
    download := job.download
    
    req, err := http.NewRequestWithContext(job.ctx, "GET", download.URL, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("User-Agent", dm.config.UserAgent)
    
    resp, err := job.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    fullPath := filepath.Join(download.Path, download.Filename)
    file, err := os.Create(fullPath)
    if err != nil {
        return err
    }
    defer file.Close()

    var reader io.Reader = resp.Body
    if dm.config.MaxSpeed > 0 {
        reader = newRateLimitedReader(resp.Body, dm.config.MaxSpeed)
    }

    buffer := make([]byte, 32*1024)
    
    for {
        select {
        case <-job.ctx.Done():
            return job.ctx.Err()
        default:
        }

        n, err := reader.Read(buffer)
        if n > 0 {
            if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
                return writeErr
            }
            atomic.AddInt64(&download.Downloaded, int64(n))
        }

        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    return nil
}

func (dm *DownloadManager) PauseDownload(id int64) error {
    dm.mutex.RLock()
    job, exists := dm.downloads[id]
    dm.mutex.RUnlock()

    if !exists {
        return fmt.Errorf("download not found")
    }

    job.cancel()
    job.download.Status = StatusPaused
    dm.updateDownload(job.download)
    dm.notifyCallbacks(job.download)

    return nil
}

func (dm *DownloadManager) CancelDownload(id int64) error {
    dm.mutex.RLock()
    job, exists := dm.downloads[id]
    dm.mutex.RUnlock()

    if exists {
        job.cancel()
    }

    // Update status in database
    download, err := storage.GetDownload(dm.db, id)
    if err != nil {
        return err
    }

    download.Status = StatusCancelled
    dm.updateDownload(download)
    dm.notifyCallbacks(download)

    // Remove partial file if exists
    fullPath := filepath.Join(download.Path, download.Filename)
    os.Remove(fullPath)

    return nil
}

func (dm *DownloadManager) GetDownloads() ([]*Download, error) {
    return storage.GetAllDownloads(dm.db)
}

func (dm *DownloadManager) updateDownload(download *Download) {
    storage.UpdateDownload(dm.db, download)
}

func (dm *DownloadManager) processQueue() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for range ticker.C {
        if atomic.LoadInt32(&dm.activeDownloads) < int32(dm.config.MaxConcurrentDownloads) {
            if download := dm.queue.Next(); download != nil {
                dm.StartDownload(download.ID)
            }
        }
    }
}

func (dm *DownloadManager) updateStats() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for range ticker.C {
        dm.mutex.RLock()
        for _, job := range dm.downloads {
            if job.download.Status == StatusDownloading {
                // Calculate progress
                if job.download.Size > 0 {
                    job.download.Progress = float64(job.download.Downloaded) / float64(job.download.Size) * 100
                }

                // Calculate speed
                now := time.Now()
                if !job.lastUpdate.IsZero() {
                    duration := now.Sub(job.lastUpdate)
                    if duration > 0 {
                        // This is a simplified speed calculation
                        // In a real implementation, you'd track downloaded bytes over time
                        job.download.Speed = int64(float64(job.download.Downloaded) / duration.Seconds())
                    }
                }
                job.lastUpdate = now

                dm.updateDownload(job.download)
                dm.notifyCallbacks(job.download)
            }
        }
        dm.mutex.RUnlock()
    }
}

func (dm *DownloadManager) AddCallback(callback func(*Download)) {
    dm.callbacks = append(dm.callbacks, callback)
}

func (dm *DownloadManager) notifyCallbacks(download *Download) {
    for _, callback := range dm.callbacks {
        go callback(download)
    }
}

// Rate-limited reader
type rateLimitedReader struct {
    reader   io.Reader
    limiter  chan time.Time
    maxBytes int64
}

func newRateLimitedReader(reader io.Reader, maxBytesPerSecond int64) *rateLimitedReader {
    limiter := make(chan time.Time, 1)
    
    go func() {
        ticker := time.NewTicker(time.Second / time.Duration(maxBytesPerSecond/1024))
        defer ticker.Stop()
        
        for t := range ticker.C {
            select {
            case limiter <- t:
            default:
            }
        }
    }()

    return &rateLimitedReader{
        reader:   reader,
        limiter:  limiter,
        maxBytes: maxBytesPerSecond,
    }
}

func (r *rateLimitedReader) Read(p []byte) (n int, err error) {
    <-r.limiter // Wait for rate limit
    return r.reader.Read(p)
}