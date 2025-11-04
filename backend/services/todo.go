package services

import (
    "context"
    "log"
    "strings"

    "github.com/anner/ai-foreign-trade-assistant/backend/domain"
    "github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// TodoService defines background processing for queued user queries.
type TodoService interface {
    Enqueue(ctx context.Context, query string) (*domain.TodoTask, error)
    ProcessNext(ctx context.Context) (bool, error)
}

type TodoServiceImpl struct {
    store     *store.Store
    enricher  EnrichmentService
    automation AutomationService
}

func NewTodoService(st *store.Store, enricher EnrichmentService, automation AutomationService) *TodoServiceImpl {
    return &TodoServiceImpl{store: st, enricher: enricher, automation: automation}
}

func (s *TodoServiceImpl) Enqueue(ctx context.Context, query string) (*domain.TodoTask, error) {
    trimmed := strings.TrimSpace(query)
    if trimmed == "" {
        return nil, nil
    }
    return s.store.CreateTodoTask(ctx, trimmed)
}

func (s *TodoServiceImpl) ProcessNext(ctx context.Context) (bool, error) {
    task, err := s.store.ClaimNextTodo(ctx)
    if err != nil || task == nil {
        return false, err
    }

    // Resolve company information
    result, err := s.enricher.ResolveCompany(ctx, &domain.ResolveCompanyRequest{Query: task.Query})
    if err != nil {
        _ = s.store.MarkTodoFailed(ctx, task.ID, err.Error())
        return true, err
    }

    // Persist to customers (idempotent create handled in store)
    name := strings.TrimSpace(result.Name)
    if name == "" {
        name = strings.TrimSpace(task.Query)
    }
    // Compose create request
    createReq := &domain.CreateCompanyRequest{
        Name:     name,
        Website:  strings.TrimSpace(result.Website),
        Country:  strings.TrimSpace(result.Country),
        Summary:  strings.TrimSpace(result.Summary),
        Contacts: result.Contacts,
    }
    customerID, err := s.store.CreateCustomer(ctx, createReq)
    if err != nil {
        // If already exists, find its ID
        if existing, _, ferr := s.store.FindCustomerByQuery(ctx, name); ferr == nil && existing != nil {
            customerID = existing.ID
        } else if existing, _, ferr := s.store.FindCustomerByQuery(ctx, result.Website); ferr == nil && existing != nil {
            customerID = existing.ID
        } else {
            _ = s.store.MarkTodoFailed(ctx, task.ID, err.Error())
            return true, err
        }
    }

    // Enqueue automation if enabled
    if s.automation != nil && customerID > 0 {
        if job, aErr := s.automation.Enqueue(ctx, customerID); aErr != nil {
            log.Printf("todo: enqueue automation failed (customer %d): %v", customerID, aErr)
        } else if job != nil {
            _ = job
        }
    }

    if err := s.store.MarkTodoCompleted(ctx, task.ID, customerID); err != nil {
        return true, err
    }
    return true, nil
}

