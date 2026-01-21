package property_amenities

import (
	"log"
	"net/http"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"
	"ollerod-pms/internal/json"
)

type handler struct {
	service Service
}

func NewHandler(svc Service) *handler {
	return &handler{
		service: svc,
	}
}

// CreatePropertyAmenity handles the creation of a new property amenity. (CRUD - Create)
func (h *handler) CreatePropertyAmenity(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a temporary CreatePropertyAmenityParams struct
	var tempAmenity repo.CreatePropertyAmenityParams

	// json.Read will decode the JSON body into the CreatePropertyAmenityParams struct
	if err := json.Read(r, &tempAmenity); err != nil {
		// If there's an error reading the request, log it and return a 400 Bad Request response
		log.Println("error reading create property amenity request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Validate the parameters
	if err := validateCreatePropertyAmenityParams(tempAmenity); err != nil {
		// If validation fails, log the error and return a 400 Bad Request response
		log.Println("error validating create property amenity params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to create the property amenity
	amenity, err := h.service.CreatePropertyAmenity(r.Context(), tempAmenity)

	// Handle errors from the service
	if err != nil {
		// If the error is ErrRelatedEntityNotFound, return a 400 Bad Request response
		if err == hf.ErrRelatedEntityNotFound {
			log.Printf("related entity not found for property amenity creation: %v", err)
			json.Write(w, http.StatusBadRequest, err)
			return
		}

		// If the error is ErrDuplicatedField, return a 409 Conflict response
		if err == hf.ErrDuplicatedField {
			log.Printf("property amenity field dupliation: %v", err)
			json.Write(w, http.StatusConflict, "property amenity already exists")
			return
		}
		log.Printf("error creating property amenity: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Write the created property amenity as a JSON response with a 201 Created status
	json.Write(w, http.StatusCreated, amenity)
}

// ListPropertyAmenities handles listing all property amenities. (CRUD - Read)
func (h *handler) ListPropertyAmenities(w http.ResponseWriter, r *http.Request) {
	amenities, err := h.service.ListPropertyAmenities(r.Context())
	if err != nil {
		if err == ErrPropertyAmenityNotFound {
			amenities = []repo.PropertyAmenity{}

			log.Println("no property amenities found")
			json.Write(w, http.StatusOK, amenities)
			return
		}
		// Log the error and return a 500 Internal Server Error response
		log.Printf("error listing property amenities: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Write the list of property amenities as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, amenities)
}

// GetPropertyAmenityById handles retrieving a property amenity by its ID. (CRUD - Read)
func (h *handler) GetPropertyAmenityById(w http.ResponseWriter, r *http.Request) {
	// Get the property amenity from the service
	amenity, err := h.service.GetPropertyAmenityById(r.Context())

	if err != nil {
		if err == ErrPropertyAmenityNotFound {
			json.Write(w, http.StatusNotFound, ErrPropertyAmenityNotFound)
			return
		}
		log.Printf("error getting property amenity by ID: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Write the property amenity as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, amenity)
}
