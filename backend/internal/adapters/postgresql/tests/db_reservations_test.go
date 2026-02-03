package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbReservations(t *testing.T) {
	ctx := context.Background()
	property := GenerateTestProperty(t, ctx)

	// Create a reservation group
	reservationGroup := GenerateTestReservationGroup(t, ctx, property.ID)

	// Create a guest
	guest := GenerateTestGuest(t, ctx, property.ID)

	travelAgent := GenerateTestTravelAgent(t, ctx, property.ID)

	insertQuery := `INSERT INTO operations.reservations (property_id, group_id, primary_guest_id, status, source, travel_agent_id) VALUES ($1, $2, $3, $4, $5, $6)`

	baseParams := TestReservation{
		PropertyID:     property.ID,
		GroupID:        nil,
		PrimaryGuestID: guest.ID,
		Status:         "confirmed",
		Source:         "website",
		TravelAgentID:  nil,
	}

	paramsSlice := []interface{}{
		baseParams.PropertyID,
		baseParams.GroupID,
		baseParams.PrimaryGuestID,
		baseParams.Status,
		baseParams.Source,
		baseParams.TravelAgentID,
	}

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		baseParams.GroupID = &reservationGroup.ID
		baseParams.TravelAgentID = &travelAgent.ID

		tests := []hf.FKExistenceTest{
			{
				Name:       "TC-RES-02 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RES-03 - Reservation Group must exist if set",
				FakeIDIdx:  1,
				RealID:     reservationGroup.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RES-04 - Primary Guest must exist",
				FakeIDIdx:  2,
				RealID:     guest.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RES-05 - Travel Agent must exist if set",
				FakeIDIdx:  5,
				RealID:     travelAgent.ID,
				BaseParams: paramsSlice,
			},
		}

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, tests)
	})

	t.Run("Code & Sequence Tests", func(t *testing.T) {
		t.Parallel()
		createdRes1 := &TestReservation{}
		err := testDB.QueryRow(context.Background(),
			`INSERT INTO operations.reservations (property_id, group_id, primary_guest_id, status, source, travel_agent_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, code, sequential`,
			baseParams.PropertyID,
			baseParams.GroupID,
			baseParams.PrimaryGuestID,
			baseParams.Status,
			baseParams.Source,
			baseParams.TravelAgentID,
		).Scan(&createdRes1.ID, &createdRes1.Code, &createdRes1.Sequential)

		assert.NoError(t, err)

		t.Run("TC-RESV-05 - Inserting a row must increment the sequential", func(t *testing.T) {
			t.Parallel()
			createdRes2 := &TestReservation{}
			err := testDB.QueryRow(context.Background(),
				`INSERT INTO operations.reservations (property_id, group_id, primary_guest_id, status, source, travel_agent_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, code, sequential`,
				baseParams.PropertyID,
				baseParams.GroupID,
				baseParams.PrimaryGuestID,
				baseParams.Status,
				baseParams.Source,
				baseParams.TravelAgentID,
			).Scan(&createdRes2.ID, &createdRes2.Code, &createdRes2.Sequential)

			assert.NoError(t, err)

			// Check that the second reservation's sequential number is exactly one greater than the first
			assert.Equal(t, createdRes1.Sequential+1, createdRes2.Sequential, "Expected sequential number to increment by 1")
		})

		t.Run("TC-RESV-06 - The code must be in format RES-XXXXXX where XXXXXX is a 6 digit number", func(t *testing.T) {
			t.Parallel()
			// Validate code format
			matched, err := hf.MatchRegex(`^RES-\d{6}$`, createdRes1.Code)
			assert.NoError(t, err, "Error matching regex: %v", err)
			assert.True(t, matched, "Reservation code does not match expected format: got %s", createdRes1.Code)
		})
	})

	t.Run("TC-RESV-10 - Notes must not exceed 2500 characters", func(t *testing.T) {
		t.Parallel()

		paramsSlice = append(paramsSlice, "REservation notes")

		notesInsertQuery := `INSERT INTO operations.reservations (property_id, group_id, primary_guest_id, status, source, travel_agent_id, notes) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		hf.RunConstraintTests(t, ctx, testDB, notesInsertQuery, paramsSlice, []hf.ConstraintTest{
			{
				Name:        "TC-RESV-10 - Notes must not exceed 2500 characters",
				Field:       "notes",
				FieldIndex:  6,
				Value:       strings.Repeat("A", 2501),
				ExpectedErr: hf.CheckViolationCode,
			},
		})
	})
}
