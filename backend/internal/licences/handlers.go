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

func (h *handler) ListLicences(w http.ResponseWriter, r *http.Request) {
	licences, err := h.service.ListLicences(r.Context())
	if err != nil {
		log.Printf("error listing licences: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, licences)
}

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
