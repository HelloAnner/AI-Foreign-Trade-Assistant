package task

import (
	"context"
	"log"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/services"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// Runner periodically polls scheduled tasks and triggers follow-up sending.
type Runner struct {
	store     *store.Store
	scheduler services.SchedulerService
	interval  time.Duration
	stopCh    chan struct{}
}

// NewRunner constructs a new follow-up runner.
func NewRunner(st *store.Store, scheduler services.SchedulerService) *Runner {
	return &Runner{
		store:     st,
		scheduler: scheduler,
		interval:  time.Minute,
		stopCh:    make(chan struct{}),
	}
}

// Start launches the background loop.
func (r *Runner) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.stopCh:
				return
			case <-ticker.C:
				r.process(ctx)
			}
		}
	}()
}

// Stop stops the runner.
func (r *Runner) Stop() {
	close(r.stopCh)
}

func (r *Runner) process(ctx context.Context) {
	tasks, err := r.store.FetchDueTasks(ctx, 5)
	if err != nil {
		log.Printf("[scheduler] 拉取任务失败: %v", err)
		return
	}
	for _, task := range tasks {
		if err := r.scheduler.RunNow(ctx, task.ID); err != nil {
			log.Printf("[scheduler] 执行任务 %d 失败: %v", task.ID, err)
		} else {
			log.Printf("[scheduler] 任务 %d 已发送", task.ID)
		}
	}
}
