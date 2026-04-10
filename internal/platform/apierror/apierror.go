package apierror

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// MapPostgresError maps a postgres error to an APIError
// Example:
//
//	err := store.SetCurrentPropertyID(ctx, "property_id")
//	if err != nil {
//		return apierror.MapPostgresError(err)
//	}
func MapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	// Handle SQLSTATE 23505 (Unique Violation) -> return a 409 Conflict
	// Handle SQLSTATE 23503 (FK Violation) -> return a 400 Bad Request
	return err
}
