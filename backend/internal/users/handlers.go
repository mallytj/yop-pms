package users

import (
	"errors"
	"log"
	"net/http"

	"ollerod-pms/internal/helpers"
	"ollerod-pms/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type handler struct {
	service Service
}

func NewHandler(svc Service) *handler {
	return &handler{
		service: svc,
	}
}

// ListUsers handles listing all users. (CRUD - Read)
func (h *handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Call the service to get the list of users
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		// Log the error and return a 500 Internal Server Error response
		log.Printf("error listing users: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Write the list of users as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, users)
}

// CreateUser handles the creation of a new user. (CRUD - Create)
func (h *handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a temporary createUserParams struct
	var tempUser createUserParams

	// json.Read will decode the JSON body into the createUserParams struct
	if err := json.Read(r, &tempUser); err != nil {
		log.Println("error reading create user request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Validate the parameters
	if err := validateCreateUserParams(&tempUser); err != nil {
		log.Println("error validating create user params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to create the user
	user, err := h.service.CreateUser(r.Context(), tempUser)
	if err != nil {
		// If the error is ErrLicenceNotFound, return a 400 Bad Request response
		if err == ErrLicenceNotFound {
			log.Println("licence not found for user creation:", err)
			json.Write(w, http.StatusBadRequest, err)
			return
		}

		if err == ErrDuplicatedField {
			log.Println("duplicated user error:", err)
			json.Write(w, http.StatusConflict, err)
			return
		}

		// Log the error and return a 500 Internal Server Error response
		log.Println("error creating user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	// Write the created user as a JSON response with a 201 Created status
	json.Write(w, http.StatusCreated, user)
}

// GetUserById handles retrieving a user by their ID. (CRUD - Read)
func (h *handler) GetUserById(w http.ResponseWriter, r *http.Request) {
	// Extract the userID from the URL parameters
	rawUserID := chi.URLParam(r, "userID")

	// Convert to UUID and validate
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		log.Println("invalid user ID format:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to get the user by ID
	user, err := h.service.GetUserById(r.Context(), userID)

	// Handle potential errors
	if err != nil {
		// If the error is ErrUserNotFound, return a 404 Not Found response
		if err == ErrUserNotFound {
			log.Println("user not found:", err)
			json.Write(w, http.StatusNotFound, err)
			return
		}

		// Log the error and return a 500 Internal Server Error response
		log.Println("error getting user by ID:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	// Write the retrieved user as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, user)
}

// UpdateUser handles updating an existing user by their ID. (CRUD - Update)
func (h *handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Println("error parsing form data:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Read request body into updateUserParams struct
	var params = &updateUserParams{}
	if err := json.Read(r, params); err != nil {
		log.Println("error reading update user request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Extract userID from URL parameters
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		log.Println("invalid user ID format:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// If userID is nil, return a 400 Bad Request response
	if userID == uuid.Nil {
		log.Println("user ID cannot be nil")
		json.Write(w, http.StatusBadRequest, errors.New("user ID cannot be nil"))
		return
	}

	// Set the UserID in params
	params.UserID = userID

	// Validate parameters
	if err := validateUpdateUserParams(params); err != nil {
		// If validation fails, return a 400 Bad Request response
		log.Println("error validating update user params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to update the user
	user, err := h.service.UpdateUser(r.Context(), params.UserID, *params)
	if err != nil {
		// If the error is ErrUserNotFound, return a 404 Not Found response
		if err == ErrUserNotFound {
			log.Println("user not found:", err)
			json.Write(w, http.StatusNotFound, err)
			return
		}

		if err == ErrDuplicatedField {
			log.Println("duplicated user error:", err)
			json.Write(w, http.StatusConflict, err)
			return
		}
		// Log the error and return a 500 Internal Server Error response
		log.Println("error updating user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	// Write the updated user as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, user)
}

// DeleteUser handles deleting a user by their ID. (CRUD - Delete)
func (h *handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Extract userID from URL parameters and parse it
	userID, err := helpers.ExtractAndParseUUIDParam(r, "userID")
	if err != nil {
		// If conversion fails, return a 400 Bad Request response
		log.Println("invalid user ID format:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to delete the user
	err = h.service.DeleteUser(r.Context(), userID)
	if err != nil {
		// If the error is ErrUserNotFound, return a 404 Not Found response
		if err == ErrUserNotFound {
			log.Println("user not found:", err)
			json.Write(w, http.StatusNotFound, err)
			return
		}

		// Log the error and return a 500 Internal Server Error response
		log.Println("error deleting user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	// Return a 204 No Content response on successful deletion
	json.Write(w, http.StatusNoContent, nil)
}

// GetLicence handles retrieving a user's licence by their user ID. (CRUD - Read)
func (h *handler) GetLicence(w http.ResponseWriter, r *http.Request) {
	// Extract userID from URL parameters and parse it
	userID, err := helpers.ExtractAndParseUUIDParam(r, "userID")
	if err != nil {
		log.Println("invalid user ID format:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to get the user's licence
	licences, err := h.service.GetLicence(r.Context(), userID)
	if err != nil {
		// If the error is ErrLicenceNotFound, return a 404 Not Found response
		if err == ErrLicenceNotFound {
			log.Println("licence not found:", err)
			json.Write(w, http.StatusNotFound, err)
			return
		}

		// Log the error and return a 500 Internal Server Error response
		log.Println("error getting licences for user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	// Write the retrieved licences as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, licences)
}
