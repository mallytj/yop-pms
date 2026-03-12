package cache

import (
	"fmt"

	"github.com/google/uuid"
)

func PlannerCacheKey(propertyID uuid.UUID, startDate, endDate string) string {
	return fmt.Sprintf("planner:%s:%s:%s", propertyID.String(), startDate, endDate)
}
