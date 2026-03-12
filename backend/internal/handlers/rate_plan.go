package handlers

import (
	"log/slog"
	"net/http"
	"ollerod-pms/internal/cache"
	"ollerod-pms/internal/json"
	"ollerod-pms/internal/service"
)

type RatePlanHandler struct {
	service service.RatePlan
	cache   *cache.CacheInvalidator
	logger  *slog.Logger
}

func NewRatePlanHandler(s service.RatePlan, c *cache.CacheInvalidator, l *slog.Logger) *RatePlanHandler {
	return &RatePlanHandler{
		service: s,
		cache:   c,
		logger:  l,
	}
}

// @Summary			Get Rate Plans
// @Description	Retrieve rate plans for pricing and resource management.
// @Tags			Pricing
// @Accept			json
// @Produce			json
// @Success			200			{object}	[]service.RatePlan 		"Successful response with rate plan"
// @Failure      	400  		{object}  	BadRequestError  		"Invalid Date Range or Format"
// @Failure      	403  		{object}  	ForbiddenError  		"User Not Authorized"
// @Failure      	500  		{object}  	InternalServerError  	"Internal Database Error"
// @Router			/rate-plans [get]
func (h *RatePlanHandler) GetRatePlans(w http.ResponseWriter, r *http.Request) {
	// TODO Cache Check
	// TODO add filtering

	ctx := r.Context()

	ratePlans, err := h.service.Get(ctx)
	if err != nil {
		h.logger.Warn("failed to retrieve rate plans", "error", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusOK, ratePlans)
}
