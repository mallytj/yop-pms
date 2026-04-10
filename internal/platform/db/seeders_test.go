package db_test

import (
	"context"

	"github.com/google/uuid"
)

func seedLicence(ctx context.Context) uuid.UUID {
	var id uuid.UUID
	row := testDB.QueryRow(ctx, `
	INSERT INTO operations.licences
	(licence_key, organisation_name, contact_email)
	VALUES ($1, $2, $3)
	RETURNING id`,
		"YOP-01010", "Yop Pms", "test@yop.com")
	if err := row.Scan(&id); err != nil {
		panic(err)
	}

	return id
}

func seedProperty(ctx context.Context, licID uuid.UUID) uuid.UUID {
	if licID == uuid.Nil {
		licID = seedLicence(ctx)
	}

	address := "123 Test St, Test Town, Test Country" + uuid.NewString()[:3]

	var id uuid.UUID
	row := testDB.QueryRow(ctx, `
	INSERT INTO operations.properties
	(name, licence_id, address, timezone)
	VALUES ($1, $2, $3, $4)	
	RETURNING id`, "YOP-01010", licID, address, "Europe/London")
	if err := row.Scan(&id); err != nil {
		panic(err)
	}

	return id
}
