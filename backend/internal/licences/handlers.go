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
	var tempLicence repo.CreateLicenceParams

	if err := json.Read(r, &tempLicence); err != nil {
		log.Println("error reading create licence request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	if err := validateCreateLicenceParams(tempLicence); err != nil {
		log.Println("error validating create licence params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	licence, err := h.service.CreateLicence(r.Context(), tempLicence)
	if err != nil {
		log.Printf("error creating licence: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusCreated, licence)
}

// ListLicences handles listing all licences. (CRUD - Read)
func (h *handler) ListLicences(w http.ResponseWriter, r *http.Request) {
	licences, err := h.service.ListLicences(r.Context())
	if err != nil {
		log.Printf("error listing licences: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, licences)
}

// GetLicenceById handles retrieving a licence by its ID. (CRUD - Read)
func (h *handler) GetLicenceById(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		http.Error(w, "invalid licence ID", http.StatusBadRequest)
		return
	}

	licence, err := h.service.GetLicenceById(r.Context(), licenceID)
	if err != nil {
		log.Printf("error getting licence by ID: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, licence)
}

func (h *handler) GetUsersByID(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))
	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		http.Error(w, "invalid licence ID", http.StatusBadRequest)
		return
	}

	users, err := h.service.GetUsersByID(r.Context(), licenceID)
	if err != nil {
		log.Printf("error getting users by licence ID: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, users)
}

// UpdateLicence handles updating an existing licence. (CRUD - Update)
func (h *handler) UpdateLicence(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		http.Error(w, "invalid licence ID", http.StatusBadRequest)
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
		log.Printf("error updating licence: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, licence)
}

// DeleteLicence handles deleting a licence. (CRUD - Delete)
func (h *handler) DeleteLicence(w http.ResponseWriter, r *http.Request) {
	licenceID := uuid.MustParse(chi.URLParam(r, "licenceID"))

	if licenceID == uuid.Nil {
		log.Println("invalid licence ID")
		http.Error(w, "invalid licence ID", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteLicence(r.Context(), licenceID)
	if err != nil {
		if err == ErrLicenceNotFound {
			http.Error(w, "licence not found", http.StatusNotFound)
			return
		}
		log.Printf("error deleting licence: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
