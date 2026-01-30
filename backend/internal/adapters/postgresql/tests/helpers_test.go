package db_tests

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
	// Create test licence
	var lic *TestLicence
	// Assign memory address
	lic = &TestLicence{}

	// Insert test licence into database
	// Use a unique licence key for each test run
	licenceKey := "YOP-" + hf.Lpad(fmt.Sprint(rand.Intn(90000+10000)), "0", 5)
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
	// Create test licence
	licence := GenerateTestLicence(t, ctx, true)

	// Get licence ID
	var property *TestProperty
	property = &TestProperty{}

	// Insert test property into database
	propertyName := "Test Property " + hf.Lpad(fmt.Sprint(rand.Intn(90000+10000)), "0", 5)
	row := testDB.QueryRow(ctx,
		`INSERT INTO operations.properties (licence_id, name, address, timezone)
				VALUES ($1, $2, $3, $4) RETURNING id, name, address, timezone`,
		licence.ID, propertyName, "123 Test St, Test City", "UTC").Scan(&property.ID, &property.name, &property.address, &property.timezone)
	assert.NoError(t, row)

	return property
}
