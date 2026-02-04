//go:build ignore
package db_tests

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

func TestSystemFlow(t *testing.T) {
	t.Run("TC-SYS-01 - Database must support goose migrations", func(t *testing.T) {
		db, err := sql.Open("pgx", testDB.Config().ConnString())
		assert.NoError(t, err)
		defer db.Close()

		err = goose.Up(db, "../migrations")
		assert.NoError(t, err)
	})
}
