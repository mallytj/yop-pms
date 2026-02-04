//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbIdentityDocs(t *testing.T) {
	ctx := context.Background()

	tGuest := GenerateTestGuest(t, ctx, uuid.Nil)

	encryptedDocNumber, err := hf.HashPassword("1234567")
	assert.NoError(t, err)

	bParams := TestIdentityDoc{
		GuestID:            tGuest.ID,
		DocType:            "passport",
		IssuingCountry:     "US",
		EncryptedDocNumber: encryptedDocNumber,
		DocImageURL:        "https://example.com/documents/passport.jpg",
	}

	paramSlice := []interface{}{
		bParams.GuestID,
		bParams.DocType,
		bParams.IssuingCountry,
		bParams.EncryptedDocNumber,
		bParams.DocImageURL,
	}

	insertQuery := `INSERT INTO identity.identity_docs 
					(guest_id, doc_type, issuing_country, encrypted_doc_number, doc_image_url) 
					VALUES ($1, $2, $3, $4, $5)`

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		testCases := []hf.ConstraintTest{
			{
				Name:        "TC-IDDOC-02 - Doc Type Required",
				FieldIndex:  1, // DocType
				Field:       "doc_type",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-IDDOC-05 - Doc Number Required",
				FieldIndex:  3, // EncryptedDocNumber
				Field:       "encrypted_doc_number",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramSlice, testCases)
	})

	t.Run("TC-IDDOC-03 - The issuing country must be in an ISO format", func(t *testing.T) {
		t.Parallel()
	})
	t.Run("TC-IDDOC-04 - The doc image url must be valid", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("TC-IDDOC-06 - The guest must exist to add identity documents", func(t *testing.T) {
		t.Parallel()
	})
	t.Run("TC-IDDOC-07 - The doc type must be from the allowed enum values", func(t *testing.T) {
		t.Parallel()

		_, err := testDB.Exec(ctx, insertQuery, bParams.GuestID, "invalid_type", bParams.IssuingCountry, bParams.EncryptedDocNumber, bParams.DocImageURL)

		assert.True(t, hf.CheckErrorCode(err, hf.InvalidTextRepresentationCode), "Expected check violation error, got: %v", err)
	})

	t.Run("TC-IDDOC-08 - The doc number must not be stored in plain text", func(t *testing.T) {
		t.Parallel()
	})
}
