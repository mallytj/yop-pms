package licences

import (
	"log"
	"net/http"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
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

// CreateLicence handles the creation of a new licence. (CRUD - Create)
func (h *handler) CreateLicence(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a temporary CreateLicenceParams struct
	var tempLicence repo.CreateLicenceParams

	// json.Read will decode the JSON body into the CreateLicenceParams struct
	if err := json.Read(r, &tempLicence); err != nil {
		// If there's an error reading the request, log it and return a 400 Bad Request response
		log.Println("error reading create licence request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Validate the parameters
	if err := validateCreateLicenceParams(tempLicence); err != nil {
		// If validation fails, log the error and return a 400 Bad Request response
		log.Println("error validating create licence params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to create the licence
	licence, err := h.service.CreateLicence(r.Context(), tempLicence)

	// Handle errors from the service
	if err != nil {
		if err == ErrDuplicatedField {
			log.Printf("licence key already exists !!!: %v", err)
			json.Write(w, http.StatusConflict, "licence key already exists")
			return
		}

		if err == ErrLicenceNotFound {
			log.Printf("related entity not found for licence creation: %v", err)
			json.Write(w, http.StatusBadRequest, err)
			return
		}
		log.Printf("error creating licence: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusCreated, licence)
}

// ListLicences handles listing all licences. (CRUD - Read)
func (h *handler) ListLicences(w http.ResponseWriter, r *http.Request) {
	licences, err := h.service.ListLicences(r.Context())
	if err != nil {
		log.Printf("error listing licences: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, licences)
}

// GetLicenceById handles retrieving a licence by its ID. (CRUD - Read)
func (h *handler) GetLicenceById(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		json.Write(w, http.StatusBadRequest, "invalid licence ID")
		return
	}

	licence, err := h.service.GetLicenceById(r.Context(), licenceID)
	if err != nil {
		if err == ErrLicenceNotFound {
			json.Write(w, http.StatusNotFound, ErrLicenceNotFound.Error())
			return
		}
		log.Printf("error getting licence by ID: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, licence)
}

func (h *handler) GetUsersByID(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))
	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		json.Write(w, http.StatusBadRequest, "invalid licence ID")
		return
	}

	users, err := h.service.GetUsersByID(r.Context(), licenceID)
	if err != nil {
		log.Printf("error getting users by licence ID: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, users)
}

// UpdateLicence handles updating an existing licence. (CRUD - Update)
func (h *handler) UpdateLicence(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		json.Write(w, http.StatusBadRequest, "invalid licence ID")
		return
	}

	var tempLicence repo.UpdateLicenceParams

	if err := json.Read(r, &tempLicence); err != nil {
		log.Println("error reading update licence request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	if err := validateUpdateLicenceParams(tempLicence); err != nil {
		log.Println("error validating update licence params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	licence, err := h.service.UpdateLicence(r.Context(), licenceID, tempLicence)
	if err != nil {
		if err == ErrLicenceNotFound {
			json.Write(w, http.StatusNotFound, "licence not found")
			return
		}
		log.Printf("error updating licence: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, licence)
}

// DeleteLicence handles deleting a licence. (CRUD - Delete)
func (h *handler) DeleteLicence(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		json.Write(w, http.StatusBadRequest, "invalid licence ID")
		return
	}

	err := h.service.DeleteLicence(r.Context(), licenceID)
	if err != nil {
		if err == ErrLicenceNotFound {
			json.Write(w, http.StatusNotFound, "licence not found")
			return
		}
		log.Printf("error deleting licence: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
