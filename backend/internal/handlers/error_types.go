package handlers

type ConflictError struct {
	Error   string `json:"error" example:"conflict_error"`
	Message string `json:"message" example:"The requested operation could not be completed due to a conflict with the current state of the resource."`
}

type NotFoundError struct {
	Error   string `json:"error" example:"not_found"`
	Message string `json:"message" example:"The requested resource was not found."`
}

type ValidationError struct {
	Error   string `json:"error" example:"validation_error"`
	Message string `json:"message" example:"One or more validation errors occurred."`
}

type UnauthorizedError struct {
	Error   string `json:"error" example:"unauthorized"`
	Message string `json:"message" example:"You are not authorized to perform this action."`
}

type ForbiddenError struct {
	Error   string `json:"error" example:"forbidden"`
	Message string `json:"message" example:"You do not have permission to access this resource."`
}

type InternalServerError struct {
	Error   string `json:"error" example:"internal_server_error"`
	Message string `json:"message" example:"An unexpected error occurred. Please try again later."`
}

type BadRequestError struct {
	Error   string `json:"error" example:"bad_request"`
	Message string `json:"message" example:"The request could not be understood or was missing required parameters."`
}