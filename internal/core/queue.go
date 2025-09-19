package core

import (
    "sync"
    "idm-go/internal/storage"
)

type Queue struct {
    items []*storage.Download
    mutex sync.RWMutex
}

func NewQueue() *Queue {
    return &Queue{
        items: make([]*storage.Download, 0),
    }
}

func (q *Queue) Add(download *storage.Download) {
    q.mutex.Lock()
    defer q.mutex.Unlock()
    
    q.items = append(q.items, download)
}

func (q *Queue) Next() *storage.Download {
    q.mutex.Lock()
    defer q.mutex.Unlock()
    
    for i, download := range q.items {
        if download.Status == storage.StatusPending {
            // Remove from queue
            q.items = append(q.items[:i], q.items[i+1:]...)
            return download
        }
    }
    
    return nil
}

func (q *Queue) Remove(id int64) {
    q.mutex.Lock()
    defer q.mutex.Unlock()
    
    for i, download := range q.items {
        if download.ID == id {
            q.items = append(q.items[:i], q.items[i+1:]...)
            break
        }
    }
}

func (q *Queue) GetAll() []*storage.Download {
    q.mutex.RLock()
    defer q.mutex.RUnlock()
    
    result := make([]*storage.Download, len(q.items))
    copy(result, q.items)
    return result
}