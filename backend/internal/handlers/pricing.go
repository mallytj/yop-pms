package handlers

import (
	"log/slog"
	"net/http"
	"ollerod-pms/internal/cache"
	hf "ollerod-pms/internal/helpers"
	"ollerod-pms/internal/json"
	"ollerod-pms/internal/service"
)

type PricingHandler struct {
	service service.Pricing
	cache   *cache.CacheInvalidator
	logger  *slog.Logger
}

func NewPricingHandler(s service.Pricing, c *cache.CacheInvalidator, l *slog.Logger) *PricingHandler {
	return &PricingHandler{
		service: s,
		cache:   c,
		logger:  l,
	}
}

// @Summary			Get Rate Map
// @Description	Retrieve rate map for pricing and resource management.
// @Tags			Pricing
// @Accept			json
// @Produce			json
// @Param			startDate	query		string	true			"Start date for the rate map in YYYY-MM-DD format"
// @Param			endDate		query		string	true			"End date for the rate map in YYYY-MM-DD format"
// @Success			200			{object}	service.RateMap 		"Successful response with rate map"
// @Failure      	400  		{object}  	BadRequestError  		"Invalid Date Range or Format"
// @Failure      	403  		{object}  	ForbiddenError  		"User Not Authorized"
// @Failure      	500  		{object}  	InternalServerError  	"Internal Database Error"
// @Router			/rate-map [get]
func (h *PricingHandler) GetRateMap(w http.ResponseWriter, r *http.Request) {
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
	// propertyID := hf.GetPropertyIDFromCtx(r.Context())

	// cachedData, found, err := h.cache.GetRateMapCache(r.Context(), propertyID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	// if err == nil && found {
	// 	json.Write(w, http.StatusOK, cachedData)
	// 	return
	// }
	
	if err != nil {
		h.logger.Warn("Failed to retrieve cached rate map", "error", err)
	}
	// Fetch from service
	rateMap, err := h.service.GetRateMap(r.Context(), startDate, endDate)
	if err != nil {
		customErr := hf.PsqlErrToCustomErr(err)
		http.Error(w, customErr.Error(), hf.CustomErrToHTTPStatus(customErr))
		return
	}

	// go func() {
	// 	ctx := context.Background() // Use a background context for caching, as it should not be tied to the request lifecycle
	// 	// Set cache with a TTL of 30 days (cache is auto-invalidated by the CacheInvalidator when relevant data changes, but we set a long TTL to optimize for repeated access to the same date ranges)
	// 	if err := h.cache.SetRateMapCache(ctx, propertyID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), rateMap, 30*24*time.Hour); err != nil {
	// 		h.logger.Warn("failed to set rate map cache", "error", err)
	// 	}
	// }()

	json.Write(w, http.StatusOK, rateMap)
}
