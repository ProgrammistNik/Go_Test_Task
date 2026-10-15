package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
	"github.com/ProgrammistNik/Go_Test_Task/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/shopspring/decimal"
)

type ExpenseService interface {
	Create(context.Context, domain.CreateExpense) (domain.Expense, error)
	Get(context.Context, int64, *int64) (domain.Expense, error)
	List(context.Context, domain.ExpenseFilter) ([]domain.Expense, error)
	Update(context.Context, int64, *int64, domain.UpdateExpense) (domain.Expense, error)
	Delete(context.Context, int64, *int64) error
}

type HealthChecker interface {
	Ping(context.Context) error
}

type Handler struct {
	service ExpenseService
	health  HealthChecker
}

type expenseRequest struct {
	UserID   int64           `json:"user_id"`
	Amount   json.RawMessage `json:"amount"`
	Category domain.Category `json:"category"`
	SpentAt  string          `json:"spent_at"`
	Comment  string          `json:"comment"`
}

type updateExpenseRequest struct {
	Amount   *json.RawMessage `json:"amount"`
	Category *domain.Category `json:"category"`
	SpentAt  *string          `json:"spent_at"`
	Comment  *string          `json:"comment"`
}

func NewHandler(service ExpenseService, health HealthChecker) *Handler {
	return &Handler{service: service, health: health}
}

func (h *Handler) Router(webDir string) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))
	router.Get("/health", h.healthCheck)
	router.Route("/api/expenses", func(router chi.Router) {
		router.Get("/", h.listExpenses)
		router.Post("/", h.createExpense)
		router.Route("/{id}", func(router chi.Router) {
			router.Get("/", h.getExpense)
			router.Put("/", h.updateExpense)
			router.Delete("/", h.deleteExpense)
		})
	})
	router.Handle("/*", http.FileServer(http.Dir(webDir)))
	return router
}

func (h *Handler) healthCheck(writer http.ResponseWriter, request *http.Request) {
	if err := h.health.Ping(request.Context()); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "database is unavailable")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listExpenses(writer http.ResponseWriter, request *http.Request) {
	filter, err := parseFilter(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	expenses, err := h.service.List(request.Context(), filter)
	if err != nil {
		writeServiceError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, expenses)
}

func (h *Handler) createExpense(writer http.ResponseWriter, request *http.Request) {
	var input expenseRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	amount, err := parseAmount(input.Amount)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid amount")
		return
	}
	spentAt, err := time.Parse(time.DateOnly, input.SpentAt)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "spent_at must use YYYY-MM-DD format")
		return
	}
	expense, err := h.service.Create(request.Context(), domain.CreateExpense{
		UserID: input.UserID, Amount: amount, Category: input.Category, SpentAt: spentAt, Comment: input.Comment,
	})
	if err != nil {
		writeServiceError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, expense)
}

func (h *Handler) getExpense(writer http.ResponseWriter, request *http.Request) {
	id, err := parseID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid expense id")
		return
	}
	expense, err := h.service.Get(request.Context(), id, nil)
	if err != nil {
		writeServiceError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, expense)
}

func (h *Handler) updateExpense(writer http.ResponseWriter, request *http.Request) {
	id, err := parseID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid expense id")
		return
	}
	var requestBody updateExpenseRequest
	if err := decodeJSON(writer, request, &requestBody); err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	input := domain.UpdateExpense{Category: requestBody.Category, Comment: requestBody.Comment}
	if requestBody.Amount != nil {
		amount, parseErr := parseAmount(*requestBody.Amount)
		if parseErr != nil {
			writeError(writer, http.StatusBadRequest, "invalid amount")
			return
		}
		input.Amount = &amount
	}
	if requestBody.SpentAt != nil {
		spentAt, parseErr := time.Parse(time.DateOnly, *requestBody.SpentAt)
		if parseErr != nil {
			writeError(writer, http.StatusBadRequest, "spent_at must use YYYY-MM-DD format")
			return
		}
		input.SpentAt = &spentAt
	}
	expense, err := h.service.Update(request.Context(), id, nil, input)
	if err != nil {
		writeServiceError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, expense)
}

func (h *Handler) deleteExpense(writer http.ResponseWriter, request *http.Request) {
	id, err := parseID(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid expense id")
		return
	}
	if err := h.service.Delete(request.Context(), id, nil); err != nil {
		writeServiceError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func parseFilter(request *http.Request) (domain.ExpenseFilter, error) {
	values := request.URL.Query()
	filter := domain.ExpenseFilter{Limit: 100}
	if value := values.Get("user_id"); value != "" {
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || userID <= 0 {
			return filter, errors.New("invalid user_id")
		}
		filter.UserID = &userID
	}
	if value := values.Get("category"); value != "" {
		category := domain.Category(value)
		if !category.Valid() {
			return filter, errors.New("invalid category")
		}
		filter.Category = &category
	}
	for value, target := range map[string]**time.Time{"from": &filter.From, "to": &filter.To} {
		if raw := values.Get(value); raw != "" {
			parsed, err := time.Parse(time.DateOnly, raw)
			if err != nil {
				return filter, errors.New(value + " must use YYYY-MM-DD format")
			}
			if value == "to" {
				parsed = parsed.AddDate(0, 0, 1)
			}
			*target = &parsed
		}
	}
	if value := values.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 500 {
			return filter, errors.New("limit must be between 1 and 500")
		}
		filter.Limit = limit
	}
	return filter, nil
}

func parseID(request *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(request, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func parseAmount(raw json.RawMessage) (decimal.Decimal, error) {
	value := strings.Trim(string(raw), "\"")
	return decimal.NewFromString(value)
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeServiceError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrExpenseNotFound):
		writeError(writer, http.StatusNotFound, "expense not found")
	case errors.Is(err, service.ErrInvalidExpense):
		writeError(writer, http.StatusBadRequest, err.Error())
	default:
		writeError(writer, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
