// Package controller — methods validate -> call service -> DTO -> respond.
// No business logic here (the Go analogue of order-service's
// *.controller.ts rule).
package controller

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/app/analytics/dto"
	"quickbite/analytics-service/app/analytics/service"
	apperr "quickbite/analytics-service/lib/errors"
	resp "quickbite/analytics-service/lib/http"
)

type AnalyticsController struct {
	service  *service.AnalyticsService
	validate *validator.Validate
}

func NewAnalyticsController(svc *service.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{service: svc, validate: validator.New()}
}

// GetRestaurantDays handles
// GET /api/v1/analytics/restaurants/:restaurantId/days?from=&to=.
func (c *AnalyticsController) GetRestaurantDays(w http.ResponseWriter, r *http.Request) error {
	restaurantID, err := c.positiveIntParam(r, "restaurantId", analytics.ErrInvalidRestaurantID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetRestaurantDays(analytics.GetRestaurantDaysInput{
		RestaurantID: restaurantID,
		From:         query.From,
		To:           query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.RestaurantDayResponsesFrom(rows))
	return nil
}

// GetRestaurantFailureRate handles
// GET /api/v1/analytics/restaurants/:restaurantId/failure-rate?from=&to=.
func (c *AnalyticsController) GetRestaurantFailureRate(w http.ResponseWriter, r *http.Request) error {
	restaurantID, err := c.positiveIntParam(r, "restaurantId", analytics.ErrInvalidRestaurantID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetRestaurantFailureRate(analytics.GetRestaurantFailureRateInput{
		RestaurantID: restaurantID,
		From:         query.From,
		To:           query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.FailureRateResponsesFrom(rows))
	return nil
}

// GetRestaurantDeliveryAvg handles
// GET /api/v1/analytics/restaurants/:restaurantId/delivery-avg?from=&to=.
func (c *AnalyticsController) GetRestaurantDeliveryAvg(w http.ResponseWriter, r *http.Request) error {
	restaurantID, err := c.positiveIntParam(r, "restaurantId", analytics.ErrInvalidRestaurantID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetRestaurantDeliveryAvg(analytics.GetRestaurantDeliveryAvgInput{
		RestaurantID: restaurantID,
		From:         query.From,
		To:           query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.DeliveryAvgResponsesFrom(rows))
	return nil
}

// GetActiveRestaurants handles GET /api/v1/analytics/restaurants/active?from=&to=.
func (c *AnalyticsController) GetActiveRestaurants(w http.ResponseWriter, r *http.Request) error {
	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetActiveRestaurants(analytics.GetActiveRestaurantsInput{
		From: query.From,
		To:   query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.ActiveRestaurantResponsesFrom(rows))
	return nil
}

// GetBranchDays handles GET /api/v1/analytics/branches/:branchId/days?from=&to=.
func (c *AnalyticsController) GetBranchDays(w http.ResponseWriter, r *http.Request) error {
	branchID, err := c.positiveIntParam(r, "branchId", analytics.ErrInvalidBranchID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetBranchDays(analytics.GetBranchDaysInput{
		BranchID: branchID,
		From:     query.From,
		To:       query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.BranchDayResponsesFrom(rows))
	return nil
}

// GetBranchFailureRate handles
// GET /api/v1/analytics/branches/:branchId/failure-rate?from=&to=.
func (c *AnalyticsController) GetBranchFailureRate(w http.ResponseWriter, r *http.Request) error {
	branchID, err := c.positiveIntParam(r, "branchId", analytics.ErrInvalidBranchID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetBranchFailureRate(analytics.GetBranchFailureRateInput{
		BranchID: branchID,
		From:     query.From,
		To:       query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.FailureRateResponsesFrom(rows))
	return nil
}

// GetProductDays handles GET /api/v1/analytics/products/:productId/days?from=&to=.
func (c *AnalyticsController) GetProductDays(w http.ResponseWriter, r *http.Request) error {
	productID, err := c.positiveIntParam(r, "productId", analytics.ErrInvalidProductID)
	if err != nil {
		return err
	}

	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetProductDays(analytics.GetProductDaysInput{
		ProductID: productID,
		From:      query.From,
		To:        query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.ProductDayResponsesFrom(rows))
	return nil
}

// GetPlatformDays handles GET /api/v1/analytics/platform/days?from=&to=.
func (c *AnalyticsController) GetPlatformDays(w http.ResponseWriter, r *http.Request) error {
	query, err := c.dateRangeQuery(r)
	if err != nil {
		return err
	}

	rows, err := c.service.GetPlatformDays(analytics.GetPlatformDaysInput{
		From: query.From,
		To:   query.To,
	})
	if err != nil {
		return err
	}

	resp.SendSuccess(w, http.StatusOK, dto.PlatformDayResponsesFrom(rows))
	return nil
}

// positiveIntParam reads a chi URL param and parses it as a positive
// int64, returning invalidErr (one of analytics.ErrInvalid*ID) on any
// failure — the one shared validation shape every :xId path param in this
// module needs.
func (c *AnalyticsController) positiveIntParam(r *http.Request, name string, invalidErr error) (int64, error) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || value <= 0 {
		return 0, invalidErr
	}
	return value, nil
}

// dateRangeQuery reads and validates the {from, to} query string shared by
// every day-range endpoint in this module.
func (c *AnalyticsController) dateRangeQuery(r *http.Request) (dto.DateRangeQuery, error) {
	query := dto.DateRangeQuery{
		From: r.URL.Query().Get("from"),
		To:   r.URL.Query().Get("to"),
	}
	if err := c.validate.Struct(query); err != nil {
		return dto.DateRangeQuery{}, apperr.New("VALIDATION_ERROR", "Invalid query parameters", http.StatusBadRequest).
			WithDetails(err.Error())
	}
	return query, nil
}
