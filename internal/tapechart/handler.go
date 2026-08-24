package tapechart

import (
	"fmt"
	"net/http"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// maxTapeChartRangeDays is the largest [from, to) window a single request
// may span
const maxTapeChartRangeDays = 90

// GetTapeChart godoc
//
//	@Summary		Get tape chart data
//	@Description	Returns room availability grid data for a date range
//	@Tags			tape-chart
//	@Produce		json
//	@Param			from		query		string	true	"Range start (YYYY-MM-DD)"
//	@Param			to			query		string	true	"Range end (YYYY-MM-DD)"
//	@Param			include		query		string	false	"shallow or full"	Enums(shallow, full)
//	@Success		200			{object}	TapeChartResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/v1/tape-chart [get]
func (h *Handler) GetTapeChart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	from, to, include, err := parseTapeChartQuery(r)
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}

	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	data, err := h.svc.GetTapeChart(ctx, propertyID, from, to, include)
	if err != nil {
		platformjson.WriteError(w, r, apierror.ErrInternal.WithMessage("internal server error"))
		return
	}

	platformjson.WriteJSON(w, http.StatusOK, TapeChartResponse{Data: data})
}

func parseTapeChartQuery(r *http.Request) (time.Time, time.Time, IncludeMode, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	includeStr := r.URL.Query().Get("include")

	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage("from and to query parameters are required")
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage("from must be YYYY-MM-DD format")
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage("to must be YYYY-MM-DD format")
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage("to must be after from")
	}

	if to.Sub(from) > maxTapeChartRangeDays*24*time.Hour {
		return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage(
			fmt.Sprintf("date range exceeds %d-day maximum", maxTapeChartRangeDays),
		)
	}

	include := IncludeFull
	if includeStr != "" {
		switch IncludeMode(includeStr) {
		case IncludeShallow, IncludeFull:
			include = IncludeMode(includeStr)
		default:
			return time.Time{}, time.Time{}, "", apierror.ErrBadRequest.WithMessage("include must be 'shallow' or 'full'")
		}
	}

	return from, to, include, nil
}
