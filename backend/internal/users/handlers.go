package users

import (
	"log"
	"net/http"

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

func (h *handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		log.Printf("error listing users: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, users)
}

func (h *handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var tempUser createUserParams

	if err := json.Read(r, &tempUser); err != nil {
		log.Println("error reading create user request:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	if err := validateCreateUserParams(&tempUser); err != nil {
		log.Println("error validating create user params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.CreateUser(r.Context(), tempUser)
	if err != nil {
		log.Println("error creating user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusCreated, user)
}

func (h *handler) GetUserById(w http.ResponseWriter, r *http.Request) {
	rawUserID := chi.URLParam(r, "userID")

	user, err := h.service.GetUserById(r.Context(), uuid.MustParse(rawUserID))
	if err != nil {
		log.Println("error getting user by ID:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusOK, user)
}

func (h *handler) UpdateUser(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		log.Println("error parsing form data:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	params := updateUserParams{
		UserID:    uuid.MustParse(chi.URLParam(r, "userID")),
		Username:  r.FormValue("username"),
		Email:     r.FormValue("email"),
		Password:  r.FormValue("password"),
		Role:      r.FormValue("role"),
		FirstName: r.FormValue("first_name"),
		LastName:  r.FormValue("last_name"),
		IsActive:  r.FormValue("is_active") == "true",
	}

	if err := validateUpdateUserParams(&params); err != nil {
		log.Println("error validating update user params:", err)
		json.Write(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.UpdateUser(r.Context(), params.UserID, params)
	if err != nil {
		log.Println("error updating user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusOK, user)
}

func (h *handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.MustParse(chi.URLParam(r, "userID"))

	err := h.service.DeleteUser(r.Context(), userID)
	if err != nil {
		log.Println("error deleting user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusNoContent, nil)
}

func (h *handler) GetLicence(w http.ResponseWriter, r *http.Request) {
	userID := uuid.MustParse(chi.URLParam(r, "userID"))

	licences, err := h.service.GetLicence(r.Context(), userID)
	if err != nil {
		log.Println("error getting licences for user:", err)
		json.Write(w, http.StatusInternalServerError, err)
		return
	}

	json.Write(w, http.StatusOK, licences)
}
