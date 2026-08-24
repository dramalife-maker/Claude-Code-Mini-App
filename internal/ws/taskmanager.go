package ws

import (
	"context"
	"sync"
)

type taskEntry struct {
	cancel context.CancelFunc
	msgID  int64
}

var taskManager struct {
	mu    sync.Mutex
	tasks map[string]*taskEntry
}

func init() {
	taskManager.tasks = make(map[string]*taskEntry)
}

func taskStart(sessionID string, cancel context.CancelFunc, msgID int64) {
	taskManager.mu.Lock()
	if old, ok := taskManager.tasks[sessionID]; ok && old.cancel != nil {
		old.cancel()
	}
	taskManager.tasks[sessionID] = &taskEntry{cancel: cancel, msgID: msgID}
	taskManager.mu.Unlock()
}

// taskEnd 只在目前登記的任務仍是呼叫者自己（msgID 相同）時才刪除，
// 避免舊任務（例如權限被拒後已被 taskCancel 取代）的延遲收尾把新任務的登記誤刪。
func taskEnd(sessionID string, msgID int64) {
	taskManager.mu.Lock()
	if e, ok := taskManager.tasks[sessionID]; ok && e.msgID == msgID {
		delete(taskManager.tasks, sessionID)
	}
	taskManager.mu.Unlock()
}

func taskIsActive(sessionID string) bool {
	taskManager.mu.Lock()
	defer taskManager.mu.Unlock()
	_, ok := taskManager.tasks[sessionID]
	return ok
}

func taskCancel(sessionID string) {
	taskManager.mu.Lock()
	e, ok := taskManager.tasks[sessionID]
	if !ok {
		taskManager.mu.Unlock()
		return
	}
	delete(taskManager.tasks, sessionID)
	taskManager.mu.Unlock()
	if e != nil && e.cancel != nil {
		e.cancel()
	}
}
