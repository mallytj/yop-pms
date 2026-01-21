package properties

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

// CreateProperty handles the creation of a new property. (CRUD - Create)
func (h *handler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	var tempProperty repo.CreatePropertyParams

	// Decode request body into createPropertyParams struct
	if err := json.Read(r, &tempProperty); err != nil {
		log.Println("error reading create property request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Valdiate the parameters
	if err := validateCreatePropertyParams(tempProperty); err != nil {
		log.Println("error validating create property params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to create the property
	property, err := h.service.CreateProperty(r.Context(), tempProperty)
	if err != nil {
		if err == hf.ErrDuplicatedField {
			log.Printf("duplicated field error: %v", err)
			json.Write(w, http.StatusConflict, "duplicated field")
			return
		}
		if err == ErrLicenceNotFound {
			log.Printf("licence not found for property creation: %v", err)
			json.Write(w, http.StatusBadRequest, err)
			return
		}
		log.Printf("error creating property: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Return the created property with a 201 Created status
	json.Write(w, http.StatusCreated, property)
}

// ListProperties handles listing all properties. (CRUD - Read)
func (h *handler) ListProperties(w http.ResponseWriter, r *http.Request) {
	// Call the service to get the list of properties
	properties, err := h.service.ListProperties(r.Context())
	if err != nil {
		if err == ErrNoPropertiesFound {
			log.Printf("no properties found: %v", err)
			json.Write(w, http.StatusNotFound, "no properties found")
			return
		}
		// Log the error and return a 500 Internal Server Error response
		log.Printf("error listing properties: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// If no properties found, return 404 Not Found
	if len(properties) == 0 {
		properties = []repo.Property{}

		// Return 404 Not Found if no properties exist
		json.Write(w, http.StatusNotFound, properties)
		return
	}

	// Write the list of properties as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, properties)
}

// GetPropertyById handles retrieving a property by its ID. (CRUD - Read)
func (h *handler) GetPropertyById(w http.ResponseWriter, r *http.Request) {
	// Call the service to get the property by ID
	property, err := h.service.GetPropertyById(r.Context())
	if err != nil {
		// Throw 404 Not Found if the property does not exist
		if err == ErrPropertyNotFound {
			log.Printf("property not found: %v", err)
			json.Write(w, http.StatusNotFound, "property not found")
			return
		}

		// Log other errors and return a 500 Internal Server Error response
		log.Printf("error getting property by ID: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Write the property as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, property)
}

// UpdateProperty handles the updating of a property. (CRUD - Update)
func (h *handler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	// Create a temporary struct to hold the update parameters
	var tempUpdate repo.UpdatePropertyParams

	// Decode the request body into the updatePropertyParams struct
	if err := json.Read(r, &tempUpdate); err != nil {
		log.Println("error reading update property request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Validate the update parameters
	if err := validateUpdatePropertyParams(tempUpdate); err != nil {
		log.Println("error validating update property params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to update the property
	updatedProperty, err := h.service.UpdateProperty(r.Context(), tempUpdate)
	if err != nil {
		// Throw 404 Not Found if the property does not exist
		if err == ErrPropertyNotFound {
			log.Printf("property not found for update: %v", err)
			json.Write(w, http.StatusNotFound, "property not found")
			return
		}

		if err == ErrNoFieldsToUpdate {
			log.Printf("no fields to update provided: %v", err)
			json.Write(w, http.StatusNotModified, "no fields to update provided")
			return
		}

		// Log other errors and return a 500 Internal Server Error response
		log.Printf("error updating property: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Write the updated property as a JSON response with a 200 OK status
	json.Write(w, http.StatusOK, updatedProperty)
}

// DeleteProperty handles the deletion of a property. (CRUD - Delete)
func (h *handler) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	// Call the service to delete the property
	err := h.service.DeleteProperty(r.Context())
	if err != nil {
		// Throw 404 Not Found if the property does not exist
		if err == ErrPropertyNotFound {
			log.Printf("property not found for deletion: %v", err)
			json.Write(w, http.StatusNotFound, "property not found")
			return
		}

		// Log other errors and return a 500 Internal Server Error response
		log.Printf("error deleting property: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Return a 204 No Content status on successful deletion
	w.WriteHeader(http.StatusNoContent)
}

// GetLicence handles retrieving the licence associated with the property. (CRUD - Read)
func (h *handler) GetLicence(w http.ResponseWriter, r *http.Request) {
	// Get the licence associated with the property from the service
	licence, err := h.service.GetLicence(r.Context())
	if err != nil {
		if err == ErrLicenceNotFound {
			log.Printf("property not found when getting licence: %v", err)
			json.Write(w, http.StatusNotFound, "property not found")
			return
		}
		log.Printf("error getting licence for property: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, licence)
}

// GetUsers handles retrieving users associated with the property. (CRUD - Read)
func (h *handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Get users associated with the property from the service
	users, err := h.service.GetUsersByID(r.Context())
	if err != nil {
		if err == ErrPropertyNotFound {
			log.Printf("property not found when getting users: %v", err)
			json.Write(w, http.StatusNotFound, "property not found")
			return
		}
		log.Printf("error getting users for property: %v", err)
		json.Write(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, users)
}

// func (h *handler) GetDailyAvailability(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetRooms(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetRatePlans(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetGuests(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetReservations(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetAmenities(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) GetRoomTypes(w http.ResponseWriter, r *http.Request) {}
