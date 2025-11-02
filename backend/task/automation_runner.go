package task

import (
	"context"
	"log"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/services"
)

// AutomationRunner continuously processes queued automation jobs one by one.
type AutomationRunner struct {
	automation services.AutomationService
	interval   time.Duration
	stopCh     chan struct{}
}

// NewAutomationRunner constructs an automation runner.
func NewAutomationRunner(automation services.AutomationService) *AutomationRunner {
	if automation == nil {
		return nil
	}
	return &AutomationRunner{
		automation: automation,
		interval:   3 * time.Second,
		stopCh:     make(chan struct{}),
	}
}

// Start launches the background polling loop.
func (r *AutomationRunner) Start(ctx context.Context) {
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

// Stop stops the runner.
func (r *AutomationRunner) Stop() {
	if r == nil {
		return
	}
	close(r.stopCh)
}

func (r *AutomationRunner) drain(ctx context.Context) {
	if r == nil || r.automation == nil {
		return
	}
	for {
		processed, err := r.automation.ProcessNext(ctx)
		if err != nil {
			log.Printf("[automation] 处理任务失败: %v", err)
			if processed {
				// 当前任务已经标记为失败，继续尝试后续排队任务。
				continue
			}
			// Claim 阶段失败，退出本轮，等待下一次 poll 再重试。
			break
		}
		if !processed {
			break
		}
	}
}
