package seeders

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaswdr/faker/v2"
)

var PropertyID = uuid.MustParse(hf.TestPropertyID) // Fixed Property ID for seeding, so we can reference it elsewhere
func (s *Seed) SeedPlannerData(ctx context.Context) error {
	// At the start of the seeding process, clear all relevant tables
	_, err := s.db.Exec(ctx, `TRUNCATE 
		pricing.booked_daily_rates,
		pricing.daily_price_grid,
		pricing.seasonal_rates,
		pricing.base_rates,
		pricing.rate_plans,
		operations.reservation_items,
		operations.reservations,
		identity.guests,
		inventory.rooms,
		inventory.room_types,
		operations.properties,
		operations.licences
		CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to truncate: %w", err)
	}

	// 1. Create a licence
	licences, err := s.seedLicences(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to seed licences: %w", err)
	}
	licence := licences[0]

	// 2. Create a property
	properties, err := s.seedProperties(ctx, 1, licence.ID)
	if err != nil {
		return fmt.Errorf("failed to seed properties: %w", err)
	}
	property := properties[0]

	// 3. Create room types
	roomTypes, err := s.seedRoomTypes(ctx, 5, property.ID)
	if err != nil {
		return fmt.Errorf("failed to seed room types: %w", err)
	}

	// 4. Create rooms
	var roomCount int
	var rooms []room
	for _, rt := range roomTypes {
		// Create rooms based on inventory count
		newRooms, err := s.seedRooms(ctx, rt.InventoryCount, rt, roomCount)
		if err != nil {
			return fmt.Errorf("failed to seed rooms: %w", err)
		}
		rooms = append(rooms, newRooms...)
		roomCount += rt.InventoryCount
	}
	// 5. Create guests
	guests, err := s.seedGuests(ctx, roomCount*100, property.ID)
	if err != nil {
		return fmt.Errorf("failed to seed guests: %w", err)
	}

	// 6. Create rate plans
	ratePlans, err := s.seedRatePlans(ctx, property.ID)
	if err != nil {
		return fmt.Errorf("failed to seed rate plans: %w", err)
	}
	fmt.Printf("Seeded %d rate plans\n", len(ratePlans))

	// 7. Create base rates
	for _, rp := range ratePlans {
		baseRates, err := s.seedBaseRates(ctx, roomTypes, *rp.ID)
		_ = baseRates
		if err != nil {
			return fmt.Errorf("failed to seed base rates: %w", err)
		}
	}

	// 6. Hard code some reservations
	var reservations []reservation
	for _, guest := range guests {
		res, err := s.seedReservations(ctx, 1, property.ID, guest.ID)
		if err != nil {
			return fmt.Errorf("failed to seed reservations: %w", err)
		}
		reservations = append(reservations, res...)
	}
	// 7. Create reservation items for those reservations
	for _, res := range reservations {
		_, err := s.seedReservationItems(ctx, 1, res.ID, rooms, roomTypes, ratePlans)
		if err != nil {
			return fmt.Errorf("failed to seed reservation items: %w", err)
		}
	}
	return nil
}

type licence struct {
	ID               uuid.UUID
	LicenceKey       string `fake:"YOP-######"`
	OrganisationName string
	ContactEmail     string
}

func (s *Seed) seedLicences(ctx context.Context, amount int) ([]licence, error) {

	insertQuery := `INSERT INTO operations.licences (id, licence_key, organisation_name, contact_email) VALUES ($1, $2, $3, $4)`
	fake := faker.New()
	licences := make([]licence, 0, amount)
	for i := 0; i < amount; i++ {
		// Create licence
		lic := licence{
			ID:               uuid.New(),
			LicenceKey:       "YOP-" + getRandomPaddedNumber(5),
			OrganisationName: fake.Company().Name(),
			ContactEmail:     fake.Internet().Email(),
		}
		_, err := s.db.Exec(ctx, insertQuery,
			lic.ID, lic.LicenceKey, lic.OrganisationName, lic.ContactEmail)
		if err != nil {
			return []licence{}, fmt.Errorf("failed to insert licence: %w", err)
		}
		licences = append(licences, lic)
	}
	return licences, nil
}

type property struct {
	ID        uuid.UUID
	LicenceID uuid.UUID
	Name      string
	Address   string
	Timezone  string
}

func (s *Seed) seedProperties(ctx context.Context, amount int, licenceID uuid.UUID) ([]property, error) {
	insertQuery := `INSERT INTO operations.properties (id, licence_id, name, address, timezone) VALUES ($1, $2, $3, $4, $5)`
	fake := faker.New()
	properties := make([]property, 0, amount)

	for i := 0; i < amount; i++ {
		// Create property
		prop := property{
			ID:        PropertyID, // Fixed Property ID§
			LicenceID: licenceID,
			Name:      fake.Company().Name() + " Hotel",
			Address:   fake.Address().Address(),
			Timezone:  "Europe/Copenhagen",
		}
		_, err := s.db.Exec(ctx, insertQuery,
			prop.ID, prop.LicenceID, prop.Name, prop.Address, prop.Timezone)
		if err != nil {
			return []property{}, fmt.Errorf("failed to insert property: %w", err)
		}
		properties = append(properties, prop)
	}
	return properties, nil
}

type roomType struct {
	ID             uuid.UUID
	PropertyID     uuid.UUID
	Name           string
	Code           string
	StdOccupancy   int
	MaxOccupancy   int
	InventoryCount int
}

func (s *Seed) seedRoomTypes(ctx context.Context, amount int, propertyID uuid.UUID) ([]roomType, error) {
	insertQuery := `INSERT INTO inventory.room_types (id, property_id, name, code, std_occupancy, max_occupancy) VALUES ($1, $2, $3, $4, $5, $6)`
	fake := faker.New()
	roomTypes := make([]roomType, 0, amount)

	for i := 0; i < amount; i++ {
		// Create room type
		rt := roomType{
			ID:             uuid.New(),
			PropertyID:     propertyID,
			Name:           fake.Beer().Style() + " Room",
			Code:           "RT-" + getRandomPaddedNumber(4),
			StdOccupancy:   rand.Intn(2) + 1,  // 1-2
			MaxOccupancy:   rand.Intn(4) + 2,  // 2-5
			InventoryCount: rand.Intn(10) + 1, // 1-10
		}
		_, err := s.db.Exec(ctx, insertQuery,
			rt.ID, rt.PropertyID, rt.Name, rt.Code, rt.StdOccupancy, rt.MaxOccupancy)
		if err != nil {
			if hf.CheckErrorCode(err, hf.UniqueViolationCode) {
				i--
				continue
			}
			return []roomType{}, fmt.Errorf("failed to insert room type: %w", err)
		}
		roomTypes = append(roomTypes, rt)
	}

	return roomTypes, nil
}

type room struct {
	ID                 uuid.UUID
	PropertyID         uuid.UUID
	RoomTypeID         uuid.UUID
	Name               string
	HousekeepingStatus string
	OccupancyStatus    string
}

func (s *Seed) seedRooms(ctx context.Context, amount int, roomType roomType, roomCount int) ([]room, error) {
	insertQuery := `INSERT INTO inventory.rooms (id, property_id, room_type_id, name, housekeeping_status, occupancy_status) VALUES ($1, $2, $3, $4, $5, $6)`
	rooms := make([]room, 0, amount)

	for i := 0; i < amount; i++ {
		// Create room
		r := room{
			ID:                 uuid.New(),
			PropertyID:         roomType.PropertyID,
			RoomTypeID:         roomType.ID,
			Name:               fmt.Sprintf("%03d", roomCount+i+1), // Room numbers like 001, 002, etc.
			HousekeepingStatus: "clean",
			OccupancyStatus:    "vacant",
		}
		_, err := s.db.Exec(ctx, insertQuery,
			r.ID, r.PropertyID, r.RoomTypeID, r.Name, r.HousekeepingStatus, r.OccupancyStatus)
		if err != nil {
			return []room{}, fmt.Errorf("failed to insert room: %w", err)
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

type guest struct {
	ID uuid.UUID

	FirstName string
	LastName  string
	Email     string
}

func (s *Seed) seedGuests(ctx context.Context, amount int, propertyID uuid.UUID) ([]guest, error) {
	insertQuery := `INSERT INTO identity.guests (id, property_id, first_name, last_name, email) VALUES ($1, $2, $3, $4, $5)`
	fake := faker.New()
	guests := make([]guest, 0, amount)

	for i := 0; i < amount; i++ {
		// Create guest
		g := guest{
			ID:        uuid.New(),
			FirstName: fake.Person().FirstName(),
			LastName:  fake.Person().LastName(),
			Email:     fake.Internet().Email(),
		}
		_, err := s.db.Exec(ctx, insertQuery,
			g.ID, propertyID, g.FirstName, g.LastName, g.Email)
		if err != nil {
			return []guest{}, fmt.Errorf("failed to insert guest: %w", err)
		}
		guests = append(guests, g)
	}
	return guests, nil
}

type reservation struct {
	ID             uuid.UUID
	PropertyID     uuid.UUID
	PrimaryGuestID uuid.UUID
	Status         string
}

func (s *Seed) seedReservations(ctx context.Context, amount int, propertyID uuid.UUID, guestID uuid.UUID) ([]reservation, error) {
	insertQuery := `INSERT INTO operations.reservations (id, property_id, primary_guest_id, status) VALUES ($1, $2, $3, $4)`
	reservations := make([]reservation, 0, amount)
	fake := faker.New()

	for i := 0; i < amount; i++ {
		// Create reservation
		r := reservation{
			ID:             uuid.New(),
			PropertyID:     propertyID,
			PrimaryGuestID: guestID,
			Status:         fake.RandomStringElement([]string{"confirmed", "checked_in", "checked_out", "hold"}),
		}
		_, err := s.db.Exec(ctx, insertQuery,
			r.ID, r.PropertyID, r.PrimaryGuestID, r.Status)
		if err != nil {
			return []reservation{}, fmt.Errorf("failed to insert reservation: %w", err)
		}
		reservations = append(reservations, r)
	}
	return reservations, nil
}

type reservationItem struct {
	ID               uuid.UUID
	PropertyID       uuid.UUID
	ReservationID    uuid.UUID
	AssignedRoomID   uuid.UUID
	BookedRoomTypeID uuid.UUID
	RatePlanID       uuid.UUID
	StayPeriod       pgtype.Range[pgtype.Timestamptz] // [check-in, check-out]
	Status           string
	BasePrice        int
}

func (s *Seed) seedReservationItems(ctx context.Context, amount int, reservationID uuid.UUID, rooms []room, roomTypes []roomType, ratePlans []ratePlan) ([]reservationItem, error) {
	insertQuery := `INSERT INTO operations.reservation_items (id, property_id, reservation_id, assigned_room_id, booked_room_type_id, stay_period, base_rate_pence) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	reservationItems := make([]reservationItem, 0, amount)
	fake := faker.New()

	for i := 0; i < amount; i++ {
		// Randomly select a room and room type
		room := rooms[rand.Intn(len(rooms))]
		roomType := roomTypes[rand.Intn(len(roomTypes))]
		ratePlan := ratePlans[rand.Intn(len(ratePlans))]
		midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC) // Get today's date at midnight UTC
		checkInDate := midnight.AddDate(0, 0, rand.Intn(365))                                                // Check-in within next 365 days
		checkOutDate := checkInDate.AddDate(0, 0, rand.Intn(5)+1)                                            // Stay of 1-5 days

		// Create reservation item
		ri := reservationItem{
			ID:               uuid.New(),
			ReservationID:    reservationID,
			AssignedRoomID:   room.ID,
			PropertyID:       room.PropertyID,
			BookedRoomTypeID: roomType.ID,
			RatePlanID:       *ratePlan.ID,
			StayPeriod: pgtype.Range[pgtype.Timestamptz]{
				Lower: pgtype.Timestamptz{
					Time:  checkInDate, // Check-in within next 10 days
					Valid: true,
				},
				Upper: pgtype.Timestamptz{
					Time:  checkOutDate, // Check-out at least 1 day after check-in
					Valid: true,
				},
				Valid:     true,
				LowerType: pgtype.Inclusive,
				UpperType: pgtype.Exclusive,
			},
			Status: fake.RandomStringElement([]string{"booked", "checked_in", "checked_out", "no_show"}),
		}
		_, err := s.db.Exec(ctx, insertQuery,
			ri.ID, ri.PropertyID, ri.ReservationID, ri.AssignedRoomID, ri.BookedRoomTypeID, ri.StayPeriod, ri.BasePrice)
		if err != nil {
			if hf.CheckErrorCode(err, hf.ExclusionViolationCode) {
				// Skip this reservation item due to exclusion violation (date overlap)
				i--
				continue
			}
			return []reservationItem{}, fmt.Errorf("failed to insert reservation item: %w", err)
		}

		for i := range hf.NumDaysBetween(checkInDate, checkOutDate) {
			newDate := checkInDate.AddDate(0, 0, i)
			q := repo.New(s.db)

			// Get rate for this day of week
			rate, err := q.GetRate(ctx, repo.GetRateParams{
				PropertyID:   room.PropertyID,
				RoomTypeID:   roomType.ID,
				RatePlanID:   *ratePlan.ID,
				CalendarDate: pgtype.Date{Time: newDate, Valid: true},
			})
			if err != nil {
				return []reservationItem{}, fmt.Errorf("failed to get rate: %w", err)
			}

			// Create a booked daily rate
			_, err = s.db.Exec(ctx, `INSERT INTO pricing.booked_daily_rates (property_id, reservation_item_id, rate_plan_id, calendar_date, base_price_pence) 
			VALUES ($1, $2, $3, $4, $5)`,
				room.PropertyID, ri.ID, ratePlan.ID, newDate, rate.PricePence)
			if err != nil {
				return []reservationItem{}, fmt.Errorf("failed to insert booked daily rate: %w", err)
			}

		}

		reservationItems = append(reservationItems, ri)
	}
	return reservationItems, nil
}

// getRandomPaddedNumber returns a random number with string length of padLength
func getRandomPaddedNumber(padLength int) string {
	max := 1
	for i := 0; i < padLength; i++ {
		max *= 10
	}
	num := rand.Intn(max)
	return fmt.Sprintf("%0*d", padLength, num)
}
