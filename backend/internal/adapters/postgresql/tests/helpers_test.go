package db_tests

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func GetRandomString(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func splitEnumName(fullName string) (string, string) {
	parts := strings.Split(fullName, ".")
	var schema, typeName string

	if len(parts) == 2 {
		schema = parts[0]
		typeName = parts[1]
	} else {
		schema = "public"
		typeName = parts[0]
	}

	return schema, typeName
}

// FieldTestCase represents a test case for field validation
// with an example value and the expected result (valid or invalid).
type FieldTestCase struct {
	example string
	result  bool
}

type TestCreatePropertyParams struct {
	LicenceID uuid.UUID
	Name      string
	Address   string
	Timezone  string
}

type TestLicence struct {
	ID               uuid.UUID `db:"id"`
	LicenceKey       string    `db:"licence_key"`
	OrganisationName string    `db:"organisation_name"`
	ContactEmail     string    `db:"contact_email"`
	IsActive         bool      `db:"is_active"`
}

// GenerateTestLicence is a helper function to create a test licence.
// t:        The testing object.
// ctx:      The context for database operations.
// isActive: Whether the licence should be active or not.
// Returns the created TestLicence.
func GenerateTestLicence(t *testing.T, ctx context.Context, isActive bool) *TestLicence {
	lic := &TestLicence{}

	licenceKey := fmt.Sprintf("YOP-%05d", rand.Intn(100000))
	row := testDB.QueryRow(ctx,
		`INSERT INTO operations.licences (licence_key, organisation_name, contact_email, is_active)
				VALUES ($1, $2, $3, $4) RETURNING id, licence_key, organisation_name, contact_email, is_active`,
		licenceKey, "Active Org", "test@test.com", isActive).Scan(&lic.ID, &lic.LicenceKey, &lic.OrganisationName, &lic.ContactEmail, &lic.IsActive)
	assert.NoError(t, row)

	return lic
}

type TestProperty struct {
	ID       uuid.UUID
	name     string
	address  string
	timezone string
}

// GenerateTestProperty is a helper function to create a test property with a valid licence.
// t:        The testing object.
// ctx:      The context for database operations.
// Returns the created property ID.
func GenerateTestProperty(t *testing.T, ctx context.Context) *TestProperty {
	licence := GenerateTestLicence(t, ctx, true)

	property := &TestProperty{}

	propertyName := "Test Property " + uuid.New().String()[:8]
	row := testDB.QueryRow(ctx,
		`INSERT INTO operations.properties (licence_id, name, address, timezone)
				VALUES ($1, $2, $3, $4) RETURNING id, name, address, timezone`,
		licence.ID, propertyName, "123 Test St, Test City", "UTC").Scan(&property.ID, &property.name, &property.address, &property.timezone)
	assert.NoError(t, row)

	return property
}

type CreateTestUser struct {
	ID           uuid.UUID
	LicenceID    string `json:"licence_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
}

// GenerateTestUser is a helper function to create a test user with a valid licence.
// t:        The testing object.
// ctx:      The context for database operations.
// Returns the created user ID.
func GenerateTestUser(t *testing.T, ctx context.Context) *CreateTestUser {
	licence := GenerateTestLicence(t, ctx, true)

	user := &CreateTestUser{}

	hashedPassword, err := hf.HashPassword("test")
	assert.NoError(t, err, "Failed to hash password: %v", err)

	suffix := uuid.New().String()[:8]

	params := CreateTestUser{
		LicenceID:    licence.ID.String(),
		Username:     "testuser_" + suffix,
		Email:        "testuser_" + suffix + "@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "Test",
		LastName:     "User",
		Role:         "admin",
		IsActive:     true,
	}

	// Insert test user into database
	row := testDB.QueryRow(
		ctx,
		`INSERT INTO auth.users (licence_id, username, email, password_hash, first_name, last_name, role, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, licence_id, username, email, password_hash, first_name, last_name, role, is_active`,
		params.LicenceID, params.Username, params.Email, params.PasswordHash,
		params.FirstName, params.LastName, params.Role, params.IsActive).Scan(
		&user.ID, &user.LicenceID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
	)
	assert.NoError(t, row, "Failed to create test user: %v", row)

	return user
}

// GenerateTestAmenitiy is a helper function to create a test amenity for a given property.
// t:        The testing object.
// ctx:      The context for database operations.
// propertyID: The ID of the property to which the amenity belongs.
// Returns the created TestAmenity.
func GenerateTestAmenity(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestAmenity {
	amenity := &TestAmenity{}

	createParams := TestAmenity{
		PropertyID:  propertyID,
		Name:        GetRandomString(10), // Random name
		ShortCode:   GetRandomString(4),  // Random short code
		Description: "Fully equipped gym",
		IsActive:    true,
	}

	// Insert test amenity into database
	row := testDB.QueryRow(ctx,
		`INSERT INTO operations.amenities (property_id, name, short_code, description, is_active)
				VALUES ($1, $2, $3, $4, $5) RETURNING id, property_id, name, short_code, description, is_active`,
		createParams.PropertyID,
		createParams.Name,
		createParams.ShortCode,
		createParams.Description,
		createParams.IsActive,
	).Scan(&amenity.ID, &amenity.PropertyID, &amenity.Name, &amenity.ShortCode, &amenity.Description, &amenity.IsActive)

	assert.NoError(t, row)

	return amenity
}

type TestAuditLog struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Action   string
	Entity   string
	EntityID uuid.UUID
	Changes  string
}

type TestAmenity struct {
	ID          uuid.UUID
	PropertyID  uuid.UUID
	Name        string
	ShortCode   string
	Description string
	IsActive    bool
}

type TestTravelAgent struct {
	ID                uuid.UUID
	PropertyID        uuid.UUID
	Name              string
	ContactEmail      string
	ContactPhone      string
	IATACode          string
	CommissionPercent float64
	AgencyNotes       string
}

type TestRoomType struct {
	ID           uuid.UUID
	PropertyID   uuid.UUID
	Name         string
	Code         string
	StdOccupancy int
	MinOccupancy int
	MaxOccupancy int
}

// GenerateTestRoomType creates a test room type for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestRoomType(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestRoomType {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	roomType := &TestRoomType{}

	err := testDB.QueryRow(ctx,
		`INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, min_occupancy, max_occupancy)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, property_id, name, code, std_occupancy, min_occupancy, max_occupancy`,
		propertyID,
		"Room Type "+uuid.New().String()[:8],
		GetRandomString(5),
		2, 1, 2,
	).Scan(
		&roomType.ID, &roomType.PropertyID, &roomType.Name, &roomType.Code,
		&roomType.StdOccupancy, &roomType.MinOccupancy, &roomType.MaxOccupancy,
	)
	assert.NoError(t, err)

	return roomType
}

type TestRoom struct {
	ID                 uuid.UUID
	PropertyID         uuid.UUID
	RoomTypeID         uuid.UUID
	Name               string
	HousekeepingStatus string
	OccupancyStatus    string
}

// GenerateTestRoom creates a test room for the given property and room type.
// If propertyID is uuid.Nil, a new property is created automatically.
// If roomTypeID is uuid.Nil, a new room type is created automatically under the property.
func GenerateTestRoom(t *testing.T, ctx context.Context, propertyID, roomTypeID uuid.UUID) *TestRoom {
	if propertyID == uuid.Nil {
		property := GenerateTestProperty(t, ctx)
		propertyID = property.ID
	}
	if roomTypeID == uuid.Nil {
		roomType := GenerateTestRoomType(t, ctx, propertyID)
		roomTypeID = roomType.ID
	}

	room := &TestRoom{}

	err := testDB.QueryRow(ctx,
		`INSERT INTO inventory.rooms (property_id, room_type_id, name)
			VALUES ($1, $2, $3)
			RETURNING id, property_id, room_type_id, name, housekeeping_status, occupancy_status`,
		propertyID,
		roomTypeID,
		"Room "+uuid.New().String()[:8],
	).Scan(
		&room.ID, &room.PropertyID, &room.RoomTypeID, &room.Name,
		&room.HousekeepingStatus, &room.OccupancyStatus,
	)
	assert.NoError(t, err)

	return room
}

type TestGuest struct {
	ID         uuid.UUID
	PropertyID uuid.UUID
	FirstName  string
	LastName   string
	Email      string
	Phone      string
}

// GenerateTestGuest is a helper function to create a test guest under a new property.

func GenerateTestGuest(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestGuest {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	guest := &TestGuest{}

	guestParams := TestGuest{
		PropertyID: propertyID,
		FirstName:  "Test",
		LastName:   "Guest",
		Email:      "test.guest@example.com",
		Phone:      "1234567890",
	}

	insertQuery := `INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number)
				VALUES ($1, $2, $3, $4, $5) RETURNING id, property_id, first_name, last_name, email, phone_number`

	// Insert test guest into database
	err := testDB.QueryRow(ctx,
		insertQuery,
		guestParams.PropertyID,
		guestParams.FirstName,
		guestParams.LastName,
		guestParams.Email,
		guestParams.Phone,
	).Scan(
		&guest.ID,
		&guest.PropertyID,
		&guest.FirstName,
		&guest.LastName,
		&guest.Email,
		&guest.Phone,
	)

	assert.NoError(t, err)

	return guest
}

// GenerateTestTravelAgent creates a test travel agent for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestTravelAgent(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestTravelAgent {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	agent := &TestTravelAgent{}

	suffix := uuid.New().String()[:8]

	err := testDB.QueryRow(ctx,
		`INSERT INTO identity.travel_agents (property_id, name, contact_email, contact_phone, agency_notes, iata_code, commission_percent)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, property_id, name, contact_email, contact_phone, iata_code, commission_percent, agency_notes`,
		propertyID,
		"Agent "+suffix,
		suffix+"@travel.com",
		"+1234567890",
		"Test agency",
		GetRandomString(3),
		10.0,
	).Scan(
		&agent.ID, &agent.PropertyID, &agent.Name, &agent.ContactEmail,
		&agent.ContactPhone, &agent.IATACode, &agent.CommissionPercent, &agent.AgencyNotes,
	)
	assert.NoError(t, err)

	return agent
}

type TestIdentityDoc struct {
	ID                 uuid.UUID
	GuestID            uuid.UUID
	DocType            string
	EncryptedDocNumber string
	DocImageURL        string
	IssuingCountry     string
	ExpiryDate         string
}

// GenerateTestIdentityDoc creates a test identity document for the given guest.
// If guestID is uuid.Nil a new guest (and its parent property) is created automatically.
func GenerateTestIdentityDoc(t *testing.T, ctx context.Context, guestID uuid.UUID) *TestIdentityDoc {
	if guestID == uuid.Nil {
		guestID = GenerateTestGuest(t, ctx, uuid.Nil).ID
	}

	encryptedDocNumber, err := hf.HashPassword(uuid.New().String()[:8])
	assert.NoError(t, err)

	doc := &TestIdentityDoc{}

	err = testDB.QueryRow(ctx,
		`INSERT INTO identity.identity_docs (guest_id, doc_type, issuing_country, encrypted_doc_number, doc_image_url)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, guest_id, doc_type, encrypted_doc_number, doc_image_url, issuing_country`,
		guestID,
		"passport",
		"US",
		encryptedDocNumber,
		"https://example.com/docs/passport.jpg",
	).Scan(
		&doc.ID, &doc.GuestID, &doc.DocType, &doc.EncryptedDocNumber,
		&doc.DocImageURL, &doc.IssuingCountry,
	)
	assert.NoError(t, err)

	return doc
}

type TestMaintenaceBlock struct {
	ID              uuid.UUID
	RoomID          uuid.UUID
	BlockPeriod     pgtype.Range[pgtype.Timestamptz] // Using a slice to represent the range
	Reason          string
	Type            string
	CreatedByUserID uuid.UUID
}

// GenerateTestMaintenanceBlock creates a test maintenance block for the given room.
// If roomID is uuid.Nil a new room (and its parent property and room type) is created automatically.
// If createdByUserID is uuid.Nil a new user (and its parent licence) is created automatically.
func GenerateTestMaintenanceBlock(t *testing.T, ctx context.Context, roomID, createdByUserID uuid.UUID) *TestMaintenaceBlock {
	if roomID == uuid.Nil {
		roomID = GenerateTestRoom(t, ctx, uuid.Nil, uuid.Nil).ID
	}
	if createdByUserID == uuid.Nil {
		createdByUserID = GenerateTestUser(t, ctx).ID
	}

	block := TestMaintenaceBlock{
		RoomID:          roomID,
		Reason:          "Routine Maintenance",
		Type:            "cleaning",
		BlockPeriod:     *hf.ToPgTstzRange(time.Now(), time.Now().Add(24*time.Hour)),
		CreatedByUserID: createdByUserID,
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO inventory.maintenance_blocks (room_id, block_period, reason, type, created_by_user_id)
			VALUES ($1, tstzrange($2, $3), $4, $5, $6) RETURNING id`,
		block.RoomID,
		block.BlockPeriod.Lower,
		block.BlockPeriod.Upper,
		block.Reason,
		block.Type,
		block.CreatedByUserID,
	).Scan(&block.ID)
	assert.NoError(t, err)

	return &block
}

type RPDerivationRule struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type TestRatePlan struct {
	ID               uuid.UUID
	PropertyID       uuid.UUID
	Name             string
	Code             string
	Description      string
	IsActive         bool
	ParentRatePlanID *uuid.UUID
	DerivationRule   *RPDerivationRule
	CurrencyCode     string
}

// GenerateTestRatePlan creates a test rate plan for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestRatePlan(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestRatePlan {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	ratePlan := TestRatePlan{
		PropertyID:       propertyID,
		Name:             "Rate Plan " + uuid.New().String()[:8],
		Code:             "RP" + uuid.New().String()[:1],
		Description:      "Test rate plan description",
		IsActive:         true,
		CurrencyCode:     "GBP",
		ParentRatePlanID: nil,
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, currency_code)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
		ratePlan.PropertyID,
		ratePlan.Name,
		ratePlan.Code,
		ratePlan.Description,
		ratePlan.IsActive,
		ratePlan.CurrencyCode,
	).Scan(&ratePlan.ID)
	assert.NoError(t, err)

	return &ratePlan
}

type TestCompanyProfile struct {
	ID                   uuid.UUID
	PropertyID           uuid.UUID
	TaxID                string
	NegotiatedRatePlanID *uuid.UUID
	CompanyName          string
	ContactEmail         string
	ContactPhone         string
	BillingAddress       string
	CompanyNotes         string
	HasCreditFacility    bool
}

// GenerateTestCompanyProfile creates a test company profile for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestCompanyProfile(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestCompanyProfile {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	companyProfile := TestCompanyProfile{
		PropertyID:        propertyID,
		TaxID:             "TAX" + uuid.New().String()[:5],
		CompanyName:       "Company " + uuid.New().String()[:8],
		ContactEmail:      "contact@" + uuid.New().String()[:5] + ".com",
		ContactPhone:      "+1234567890",
		BillingAddress:    "123 Business St, Commerce City",
		CompanyNotes:      "This is a test company profile.",
		HasCreditFacility: true,
	}

	query := `INSERT INTO identity.company_profiles (property_id, tax_id, company_name, contact_email, contact_phone, billing_address
			  , company_notes, has_credit_facility)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`

	err := testDB.QueryRow(ctx, query,
		companyProfile.PropertyID,
		companyProfile.TaxID,
		companyProfile.CompanyName,
		companyProfile.ContactEmail,
		companyProfile.ContactPhone,
		companyProfile.BillingAddress,
		companyProfile.CompanyNotes,
		companyProfile.HasCreditFacility,
	).Scan(&companyProfile.ID)
	assert.NoError(t, err, "Error creating test company profile")

	return &companyProfile
}

type TestDailyPriceGrid struct {
	ID                uuid.UUID
	PropertyID        uuid.UUID
	RoomTypeID        uuid.UUID
	RatePlanID        uuid.UUID
	CalendarDate      string // in YYYY-MM-DD format
	BasePricePence    int
	MinLOSRestriction int
	MaxLOSRestriction int
	IsAvailable       bool
}

// GenerateTestDailyPriceGrid creates a test daily price grid entry for the given property, room type, and rate plan.
// If propertyID is uuid.Nil a new property is created automatically.
// If roomTypeID is uuid.Nil a new room type is created automatically under the property.
// If ratePlanID is uuid.Nil a new rate plan is created automatically under the property.
func GenerateTestDailyPriceGrid(t *testing.T, ctx context.Context, propertyID, roomTypeID, ratePlanID uuid.UUID, calendarDate string) *TestDailyPriceGrid {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}
	if roomTypeID == uuid.Nil {
		roomTypeID = GenerateTestRoomType(t, ctx, propertyID).ID
	}
	if ratePlanID == uuid.Nil {
		ratePlanID = GenerateTestRatePlan(t, ctx, propertyID).ID
	}

	priceGrid := TestDailyPriceGrid{
		PropertyID:        propertyID,
		RoomTypeID:        roomTypeID,
		RatePlanID:        ratePlanID,
		CalendarDate:      calendarDate,
		BasePricePence:    10000, // £100.00
		MinLOSRestriction: 1,
		MaxLOSRestriction: 30,
		IsAvailable:       true,
	}

	query := `INSERT INTO pricing.daily_price_grid (property_id, room_type_id, rate_plan_id, calendar_date, base_price_pence, min_los_restriction, max_los_restriction, is_available)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`

	err := testDB.QueryRow(ctx, query,
		priceGrid.PropertyID,
		priceGrid.RoomTypeID,
		priceGrid.RatePlanID,
		priceGrid.CalendarDate,
		priceGrid.BasePricePence,
		priceGrid.MinLOSRestriction,
		priceGrid.MaxLOSRestriction,
		priceGrid.IsAvailable,
	).Scan(&priceGrid.ID)
	assert.NoError(t, err)

	return &priceGrid
}

type TestTaxRule struct {
	ID             uuid.UUID
	PropertyID     uuid.UUID
	Name           string
	Description    string
	TaxPercentage  float64
	IsTaxInclusive bool
}

// GenerateTestTaxRule creates a test tax rule for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestTaxRule(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestTaxRule {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	taxRule := TestTaxRule{
		PropertyID:     propertyID,
		Name:           "Tax Rule " + uuid.New().String()[:8],
		Description:    "Test tax rule description",
		TaxPercentage:  10.00,
		IsTaxInclusive: false,
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO finance.tax_rules (property_id, name, description, tax_percentage, is_tax_inclusive)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
		taxRule.PropertyID,
		taxRule.Name,
		taxRule.Description,
		taxRule.TaxPercentage,
		taxRule.IsTaxInclusive,
	).Scan(&taxRule.ID)
	assert.NoError(t, err)

	return &taxRule
}

type TestLedgerCode struct {
	ID          uuid.UUID
	PropertyID  uuid.UUID
	Code        string
	Description string
	TaxRuleID   uuid.UUID
}

// GenerateTestLedgerCode creates a test ledger code for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
// If taxRuleID is uuid.Nil a new tax rule is created automatically under the property.
func GenerateTestLedgerCode(t *testing.T, ctx context.Context, propertyID, taxRuleID uuid.UUID) *TestLedgerCode {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}
	if taxRuleID == uuid.Nil {
		taxRuleID = GenerateTestTaxRule(t, ctx, propertyID).ID
	}

	ledgerCode := TestLedgerCode{
		PropertyID:  propertyID,
		Code:        "LC" + uuid.New().String()[:5],
		Description: "Test ledger code description",
		TaxRuleID:   taxRuleID,
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO finance.ledger_codes (property_id, code, description, tax_rule)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
		ledgerCode.PropertyID,
		ledgerCode.Code,
		ledgerCode.Description,
		ledgerCode.TaxRuleID,
	).Scan(&ledgerCode.ID)
	assert.NoError(t, err)

	return &ledgerCode
}

type TestSLAccount struct {
	ID               uuid.UUID
	PropertyID       uuid.UUID
	CompanyProfileID uuid.UUID
	Name             string
	Code             string
	PaymentTermDays  int
	CreditLimitPence int
}

// GenerateTestSLAccount creates a test sales ledger account for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
// If companyProfileID is uuid.Nil a new company profile is created automatically under the property.
func GenerateTestSLAccount(t *testing.T, ctx context.Context, propertyID, companyProfileID uuid.UUID) *TestSLAccount {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}
	if companyProfileID == uuid.Nil {
		companyProfileID = GenerateTestCompanyProfile(t, ctx, propertyID).ID
	}

	account := TestSLAccount{
		PropertyID:       propertyID,
		CompanyProfileID: companyProfileID,
		Name:             "SL Account " + uuid.New().String()[:8],
		Code:             "SLA" + uuid.New().String()[:5],
		PaymentTermDays:  30,
		CreditLimitPence: 500000, // £5000.00
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO sales_ledgers.accounts (property_id, company_profile_id, name, code, payment_terms_days, credit_limit_pence)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
		account.PropertyID,
		account.CompanyProfileID,
		account.Name,
		account.Code,
		account.PaymentTermDays,
		account.CreditLimitPence,
	).Scan(&account.ID)
	assert.NoError(t, err)

	return &account
}

type TestReservationGroup struct {
	ID            uuid.UUID
	PropertyID    uuid.UUID
	MasterFolioID uuid.UUID
	Sequential    int
	Code          string
	Name          string
	Notes         string
}

// GenerateTestReservationGroup creates a test reservation group for the given property.
// If propertyID is uuid.Nil a new property is created automatically.
func GenerateTestReservationGroup(t *testing.T, ctx context.Context, propertyID uuid.UUID) *TestReservationGroup {
	if propertyID == uuid.Nil {
		propertyID = GenerateTestProperty(t, ctx).ID
	}

	group := TestReservationGroup{
		PropertyID: propertyID,
		Name:       "Reservation Group " + uuid.New().String()[:8],
		Notes:      "Test reservation group notes",
	}

	err := testDB.QueryRow(ctx,
		`INSERT INTO operations.reservation_groups (property_id, name, notes)
			VALUES ($1, $2, $3)
			RETURNING id, sequential`,
		group.PropertyID,
		group.Name,
		group.Notes,
	).Scan(&group.ID, &group.Sequential)
	assert.NoError(t, err)

	return &group
}

type TestReservation struct {
	ID             uuid.UUID
	PropertyID     uuid.UUID
	PrimaryGuestID uuid.UUID
	GroupID        *uuid.UUID
	Sequential     int
	Code           string
	Source         string
	TravelAgentID  *uuid.UUID
	Notes          string
	Status         string
}
