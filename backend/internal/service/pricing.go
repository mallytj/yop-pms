package service

import (
	"context"
	"fmt"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pricing interface {
	GetRateMap(ctx context.Context, start, end time.Time) (*RateMap, error)
}

type pricingService struct {
	*Svc
}

func NewPricingService(r repo.Queries, db *pgxpool.Pool) Pricing {
	return &pricingService{
		&Svc{
			repo: &r,
			db:   db,
		},
	}
}

type Rate struct {
	CalendarDate time.Time `json:"calendar_date"`
	RoomTypeID   uuid.UUID `json:"room_type_id"`
	RatePlanID   uuid.UUID `json:"rate_plan_id"`
	Price        int       `json:"price"`
	MinLos       int       `json:"min_los"`
	MaxLos       int       `json:"max_los"`
	Source       string    `json:"source"`
}

type RateMap struct {
	CheckInDate  string `json:"check_in_date"`
	CheckOutDate string `json:"check_out_date"`
	Rates        []Rate `json:"rates"`
}

func (s *pricingService) GetRateMap(ctx context.Context, start, end time.Time) (*RateMap, error) {
	rows, err := ExecuteTx(s.Svc, ctx, func(qtx *repo.Queries) ([]repo.GetRatesForRangeRow, error) {
		return qtx.GetRatesForRange(ctx, repo.GetRatesForRangeParams{
			StartDate:  pgtype.Date{Time: start, Valid: true},
			EndDate:    pgtype.Date{Time: end, Valid: true},
			PropertyID: hf.GetPropertyIDFromCtx(ctx),
		})
	})

	if err != nil {
		return nil, hf.PsqlErrToCustomErr(err)
	}

	rateMap := RateMap{
		CheckInDate:  start.Format("2006-01-02"),
		CheckOutDate: end.Format("2006-01-02"),
		Rates:        make([]Rate, 0),
	}

	for _, row := range rows {
		if row.RoomTypeID.Valid && row.RatePlanID.Valid {
			rateMap.Rates = append(rateMap.Rates, mapRowToRate(row))
			continue
		}
		fmt.Printf("Skipping row %v - invalid room type or rate plan", row)
	}

	return &rateMap, nil
}

func mapRowToRate(row repo.GetRatesForRangeRow) Rate {
	return Rate{
		CalendarDate: row.CalendarDate.Time,
		RoomTypeID:   row.RoomTypeID.UUID,
		RatePlanID:   row.RatePlanID.UUID,
		Price:        int(row.PricePence),
		MinLos:       int(row.MinLos),
		MaxLos:       int(row.MaxLos),
		Source:       row.Source,
	}
}
