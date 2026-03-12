package handlers

import (
	"fmt"
	"net/http"
	hf "ollerod-pms/internal/helpers"
	"ollerod-pms/internal/json"
	"ollerod-pms/internal/service"
	"ollerod-pms/internal/validator"

	"github.com/google/uuid"
)

type ReservationHandler struct {
	service service.Reservation
}

func NewReservationHandler(s service.Reservation) *ReservationHandler {
	return &ReservationHandler{
		service: s,
	}
}

type CreateReservationRequest struct {
	PrimaryGuestID uuid.UUID `json:"primary_guest_id" validate:"required"`
	

}

// @Summary Create Reservation Item
// @Description Create a new reservation item.
// @Tags Reservations
// @Accept json
// @Produce json
// @Param createData body service.CreateReservationItemData true "Data for creating the reservation item"
// @Success 200 {object} service.CreateReservationItemData "Successful response with planner data"
// @Failure 400 {object} BadRequestError "Invalid Date Range or Format"
// @Failure 403 {object} ForbiddenError "User Not Authorized"
// @Failure 500 {object} InternalServerError "Internal Database Error"
// @Router /reservation_item [post]
func (h *ReservationHandler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	// 1. Create / get guest
	// 2. Get rate plan
	// 3. Get daily price
	// 4. Assert availability
	// 5. Create reservation
	// 6. Create reservation item(s)
	// 7. Create daily prices
	// 8. FUTURE Send webhook or something to notify managers or pop up on ui or sometihng
}

// @Summary			Update Reservation Item
// @Description		Update details of a reservation item, such as assigned room, booked room type, check-in/check-out dates, or status. This allows for dynamic adjustments to reservations based on changes in availability or guest requests.
// @Tags			Reservations
// @Accept			json
// @Produce			json
// @Param			updateData		body		service.UpdateReservationItemData	true	"Data for updating the reservation item"
// @Success			200				{object}	service.UpdateReservationItemData 			"Successful response with planner data"
// @Failure      	400  			{object}  	BadRequestError  							"Invalid Date Range or Format"
// @Failure      	403  			{object}  	ForbiddenError  							"User Not Authorized"
// @Failure      	500  			{object}  	InternalServerError  						"Internal Database Error"
// @Router			/reservation_item/{reservationID} [put]
func (h *ReservationHandler) UpdateReservationItem(w http.ResponseWriter, r *http.Request) {
	// Get data from body
	var updateData service.UpdateReservationItemData
	if err := json.Read(r, &updateData); err != nil {
		json.Write(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}
	fmt.Printf("Received update request for reservation item with data: %+v\n", updateData)

	// 1. Validate input data
	if err := validator.ValidateStruct(updateData); err != nil {
		http.Error(w, "Validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Call service to update reservation item
	updatedItem, err := h.service.UpdateItem(r.Context(), updateData)
	if err != nil {
		http.Error(w, "Failed to update reservation item: "+err.Error(), hf.CustomErrToHTTPStatus(err))
		return
	}

	json.Write(w, http.StatusOK, updatedItem)
}
