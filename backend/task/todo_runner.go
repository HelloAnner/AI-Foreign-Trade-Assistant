package task

import (
    "context"
    "log"
    "time"

    "github.com/anner/ai-foreign-trade-assistant/backend/services"
)

// TodoRunner drains queued todo tasks one by one.
type TodoRunner struct {
    todo     services.TodoService
    interval time.Duration
    stopCh   chan struct{}
}

func NewTodoRunner(todo services.TodoService) *TodoRunner {
    if todo == nil {
        return nil
    }
    return &TodoRunner{todo: todo, interval: 2 * time.Second, stopCh: make(chan struct{})}
}

func (r *TodoRunner) Start(ctx context.Context) {
    if r == nil {
        return
    }
    go func() {
        ticker := time.NewTicker(r.interval)
        defer ticker.Stop()
        for {
            r.drain(ctx)
            select {
            case <-ctx.Done():
                return
            case <-r.stopCh:
                return
            case <-ticker.C:
            }
        }
    }()
}

func (r *TodoRunner) Stop() {
    if r == nil {
        return
    }
    close(r.stopCh)
}

func (r *TodoRunner) drain(ctx context.Context) {
    if r == nil || r.todo == nil {
        return
    }
    for {
        processed, err := r.todo.ProcessNext(ctx)
        if err != nil {
            log.Printf("[todo] 处理任务失败: %v", err)
            if processed {
                continue
            }
            break
        }
        if !processed {
            break
        }
    }
}

