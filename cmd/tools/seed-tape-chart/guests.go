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
		// @AI - This is less clean
		guestID, err := insertGuest(ctx, conn, propertyID, faker, i)
		if err != nil {
			return nil, err
		}
		guestIDs[i] = guestID
	}
	return guestIDs, nil
}

// insertGuest seeds one fake guest; the index suffix keeps emails unique
// across reseeds regardless of faker output.
func insertGuest(ctx context.Context, conn *pgx.Conn, propertyID string, faker *gofakeit.Faker, index int) (string, error) {
	firstName := faker.FirstName()
	lastName := faker.LastName()
	email := fmt.Sprintf("%s.%s.%d@example.com", strings.ToLower(firstName), strings.ToLower(lastName), index)
	phone := faker.Phone()

	var guestID string
	err := conn.QueryRow(ctx, `
		INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, propertyID, firstName, lastName, email, phone).Scan(&guestID)
	if err != nil {
		return "", fmt.Errorf("guest %s %s: %w", firstName, lastName, err)
	}
	return guestID, nil
}
