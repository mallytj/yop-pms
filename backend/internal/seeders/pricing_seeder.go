package seeders

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type derivationRule struct {
	Type  string
	Value int
}
type ratePlan struct {
	ID               *uuid.UUID
	PropertyID       uuid.UUID
	ParentRatePlanID uuid.UUID
	Name             string
	Code             string
	Description      string
	DerivationRule   derivationRule
	CurrencyCode     string
	IsActive         bool
}

type baseRate struct {
	ID           uuid.UUID
	RatePlanID   uuid.UUID
	RoomTypeID   uuid.UUID
	DayOfTheWeek int
	Price        int
}

func (s *Seed) seedRatePlans(ctx context.Context, propertyID uuid.UUID) ([]ratePlan, error) {
	params := []ratePlan{
		{
			PropertyID:  propertyID,
			Name:        "Room Only",
			Code:        "RO",
			Description: "Just the room",
		},
		{
			PropertyID:  propertyID,
			Name:        "Bed & Breakfast",
			Code:        "BB",
			Description: "Bed + Breakfast",
		},
		{
			PropertyID:  propertyID,
			Name:        "Cozy Package",
			Code:        "COZY",
			Description: "DBB, Tea",
		},
	}
	var ratePlans []ratePlan

	insertQuery := `INSERT INTO pricing.rate_plans (property_id, parent_rate_plan_id, name, code, description) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	for _, rp := range params {
		var ratePlanID *uuid.UUID
		err := s.db.QueryRow(ctx, insertQuery,
			rp.PropertyID, nil, rp.Name, rp.Code, rp.Description,
		).Scan(&ratePlanID)

		if err != nil {
			return ratePlans, fmt.Errorf("failed to insert rate plan: %w", err)
		}
		ratePlans = append(ratePlans, ratePlan{
			ID:               ratePlanID,
			PropertyID:       rp.PropertyID,
			ParentRatePlanID: rp.ParentRatePlanID,
			Name:             rp.Name,
			Code:             rp.Code,
			Description:      rp.Description,
			DerivationRule:   rp.DerivationRule,
			CurrencyCode:     rp.CurrencyCode,
			IsActive:         rp.IsActive,
		})
	}
	return ratePlans, nil
}

func (s *Seed) seedBaseRates(ctx context.Context, roomTypes []roomType, ratePlanID uuid.UUID) ([]baseRate, error) {
	insertQuery := `INSERT INTO pricing.base_rates (property_id, room_type_id, rate_plan_id, day_of_week, base_price_pence) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	baseRates := []baseRate{}

	for j, rt := range roomTypes {
		for i := 0; i < 7; i++ {
			price := 16500 + j*5000

			// On weekend
			if i >= 4 {
				price = 20000 + j*5000
			}

			tempRate := baseRate{
				RoomTypeID:   rt.ID,
				RatePlanID:   ratePlanID,
				DayOfTheWeek: i,
				Price:        price,
			}

			err := s.db.QueryRow(ctx, insertQuery,
				rt.PropertyID, rt.ID, ratePlanID, i, price,
			).Scan(&tempRate.ID)

			if err != nil {
				return baseRates, fmt.Errorf("failed to insert base rate: %w", err)
			}

			baseRates = append(baseRates, tempRate)
		}
	}
	return baseRates, nil
}
