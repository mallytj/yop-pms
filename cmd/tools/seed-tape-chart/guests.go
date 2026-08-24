package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5"
)

// guestCount is generous enough to avoid guests repeating too often across
// the ±3 month reservation window.
const guestCount = 60

func seedGuests(ctx context.Context, conn *pgx.Conn, propertyID string) ([]string, error) {
	faker := gofakeit.New(42)

	guestIDs := make([]string, guestCount)
	for i := range guestCount {
		firstName := faker.FirstName()
		lastName := faker.LastName()
		email := fmt.Sprintf("%s.%s.%d@example.com", strings.ToLower(firstName), strings.ToLower(lastName), i)
		phone := faker.Phone()

		err := conn.QueryRow(ctx, `
			INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, propertyID, firstName, lastName, email, phone).Scan(&guestIDs[i])
		if err != nil {
			return nil, fmt.Errorf("guest %s %s: %w", firstName, lastName, err)
		}
	}
	return guestIDs, nil
}
