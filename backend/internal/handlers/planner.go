package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"ollerod-pms/internal/cache"
	hf "ollerod-pms/internal/helpers"
	"ollerod-pms/internal/json"
	"ollerod-pms/internal/service"
	"time"
)

type PlannerHandler struct {
	service service.Planner
	cache   *cache.CacheInvalidator
	logger  *slog.Logger
}

func NewPlannerHandler(s service.Planner, c *cache.CacheInvalidator, l *slog.Logger) *PlannerHandler {
	return &PlannerHandler{
		service: s,
		cache:   c,
		logger:  l,
	}
}

// @Summary			Get Planner Data
// @Description	Retrieve planner data for scheduling and resource management.
// @Tags			Planner
// @Accept			json
// @Produce			json
// @Param			startDate	query		string	true			"Start date for the planner data in YYYY-MM-DD format"
// @Param			endDate		query		string	true			"End date for the planner data in YYYY-MM-DD format"
// @Success			200			{object}	service.PlannerData 	"Successful response with planner data"
// @Failure      	400  		{object}  	BadRequestError  		"Invalid Date Range or Format"
// @Failure      	403  		{object}  	ForbiddenError  		"User Not Authorized"
// @Failure      	500  		{object}  	InternalServerError  	"Internal Database Error"
// @Router			/planner [get]
func (h *PlannerHandler) GetPlannerData(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	start := r.URL.Query().Get("startDate")
	end := r.URL.Query().Get("endDate")

	// Validate and parse dates
	startDate, endDate, err := hf.ParseDateRange(start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check cache first
	propertyID := hf.GetPropertyIDFromCtx(r.Context())

	cachedData, found, err := h.cache.GetPlannerCache(r.Context(), propertyID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err == nil && found {
		json.Write(w, http.StatusOK, cachedData)
		return
	}
	if err != nil {
		h.logger.Warn("Failed to retrieve cached planner data", "error", err)
	}
	// Fetch from service
	plannerData, err := h.service.GetPlannerData(r.Context(), startDate, endDate)
	if err != nil {
		h.handlePlannerDataError(w, err)
		return
	}

	go func() {
		ctx := context.Background() // Use a background context for caching, as it should not be tied to the request lifecycle
		// Set cache with a TTL of 30 days (cache is auto-invalidated by the CacheInvalidator when relevant data changes, but we set a long TTL to optimize for repeated access to the same date ranges)
		if err := h.cache.SetPlannerCache(ctx, propertyID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), plannerData, 30*24*time.Hour); err != nil {
			h.logger.Warn("failed to set planner cache", "error", err)
		}
	}()

	json.Write(w, http.StatusOK, plannerData)
}

func (h *PlannerHandler) handlePlannerDataError(w http.ResponseWriter, err error) {
	switch err {
	case hf.ErrNotPermitted:
		http.Error(w, "Not permitted to access planner data", http.StatusForbidden)
	default:
		http.Error(w, "Failed to retrieve planner data", http.StatusInternalServerError)
	}
}
