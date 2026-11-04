package dto

import analytics "quickbite/analytics-service/app/analytics"

// RestaurantDayResponse is the wire shape for one row of
// GET /analytics/restaurants/:restaurantId/days — controllers never leak
// analytics.RestaurantDayRow (a service-layer type) directly.
type RestaurantDayResponse struct {
	Date          string `json:"date"`
	OrdersCount   int64  `json:"ordersCount"`
	RevenueMinor  int64  `json:"revenueMinor"`
	Currency      string `json:"currency"`
	AvgOrderMinor int64  `json:"avgOrderMinor"`
}

func RestaurantDayResponseFrom(row analytics.RestaurantDayRow) RestaurantDayResponse {
	return RestaurantDayResponse{
		Date:          row.Date,
		OrdersCount:   row.OrdersCount,
		RevenueMinor:  row.RevenueMinor,
		Currency:      row.Currency,
		AvgOrderMinor: row.AvgOrderMinor,
	}
}

func RestaurantDayResponsesFrom(rows []analytics.RestaurantDayRow) []RestaurantDayResponse {
	out := make([]RestaurantDayResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, RestaurantDayResponseFrom(row))
	}
	return out
}

// BranchDayResponse is the wire shape for GET /analytics/branches/:id/days.
type BranchDayResponse struct {
	Date          string `json:"date"`
	OrdersCount   int64  `json:"ordersCount"`
	RevenueMinor  int64  `json:"revenueMinor"`
	Currency      string `json:"currency"`
	AvgOrderMinor int64  `json:"avgOrderMinor"`
}

func BranchDayResponsesFrom(rows []analytics.BranchDayRow) []BranchDayResponse {
	out := make([]BranchDayResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, BranchDayResponse{
			Date:          row.Date,
			OrdersCount:   row.OrdersCount,
			RevenueMinor:  row.RevenueMinor,
			Currency:      row.Currency,
			AvgOrderMinor: row.AvgOrderMinor,
		})
	}
	return out
}

// ProductDayResponse is the wire shape for GET /analytics/products/:id/days.
type ProductDayResponse struct {
	Date              string `json:"date"`
	OrdersCount       int64  `json:"ordersCount"`
	QuantitySum       int64  `json:"quantitySum"`
	RevenueMinor      int64  `json:"revenueMinor"`
	Currency          string `json:"currency"`
	AvgUnitPriceMinor int64  `json:"avgUnitPriceMinor"`
}

func ProductDayResponsesFrom(rows []analytics.ProductDayRow) []ProductDayResponse {
	out := make([]ProductDayResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ProductDayResponse{
			Date:              row.Date,
			OrdersCount:       row.OrdersCount,
			QuantitySum:       row.QuantitySum,
			RevenueMinor:      row.RevenueMinor,
			Currency:          row.Currency,
			AvgUnitPriceMinor: row.AvgUnitPriceMinor,
		})
	}
	return out
}

// PlatformDayResponse is the wire shape for GET /analytics/platform/days.
type PlatformDayResponse struct {
	Date                   string  `json:"date"`
	OrdersCount            int64   `json:"ordersCount"`
	RevenueMinor           int64   `json:"revenueMinor"`
	Currency               string  `json:"currency"`
	AvgOrderMinor          int64   `json:"avgOrderMinor"`
	OrdersRejectedCount    int64   `json:"ordersRejectedCount"`
	FailureRate            float64 `json:"failureRate"`
	DeliveredCount         int64   `json:"deliveredCount"`
	AvgDeliveryMs          int64   `json:"avgDeliveryMs"`
	PaymentsCompletedCount int64   `json:"paymentsCompletedCount"`
	PaymentsCompletedSum   int64   `json:"paymentsCompletedSum"`
}

func PlatformDayResponsesFrom(rows []analytics.PlatformDayRow) []PlatformDayResponse {
	out := make([]PlatformDayResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, PlatformDayResponse{
			Date:                   row.Date,
			OrdersCount:            row.OrdersCount,
			RevenueMinor:           row.RevenueMinor,
			Currency:               row.Currency,
			AvgOrderMinor:          row.AvgOrderMinor,
			OrdersRejectedCount:    row.OrdersRejectedCount,
			FailureRate:            row.FailureRate,
			DeliveredCount:         row.DeliveredCount,
			AvgDeliveryMs:          row.AvgDeliveryMs,
			PaymentsCompletedCount: row.PaymentsCompletedCount,
			PaymentsCompletedSum:   row.PaymentsCompletedSum,
		})
	}
	return out
}

// FailureRateResponse is the wire shape for both the restaurant- and
// branch-scoped failure-rate endpoints.
type FailureRateResponse struct {
	Date                string  `json:"date"`
	OrdersCount         int64   `json:"ordersCount"`
	OrdersRejectedCount int64   `json:"ordersRejectedCount"`
	FailureRate         float64 `json:"failureRate"`
}

func FailureRateResponsesFrom(rows []analytics.FailureRateRow) []FailureRateResponse {
	out := make([]FailureRateResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, FailureRateResponse{
			Date:                row.Date,
			OrdersCount:         row.OrdersCount,
			OrdersRejectedCount: row.OrdersRejectedCount,
			FailureRate:         row.FailureRate,
		})
	}
	return out
}

// DeliveryAvgResponse is the wire shape for the restaurant-scoped
// delivery-avg endpoint.
type DeliveryAvgResponse struct {
	Date           string `json:"date"`
	DeliveredCount int64  `json:"deliveredCount"`
	AvgDeliveryMs  int64  `json:"avgDeliveryMs"`
}

func DeliveryAvgResponsesFrom(rows []analytics.DeliveryAvgRow) []DeliveryAvgResponse {
	out := make([]DeliveryAvgResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, DeliveryAvgResponse{
			Date:           row.Date,
			DeliveredCount: row.DeliveredCount,
			AvgDeliveryMs:  row.AvgDeliveryMs,
		})
	}
	return out
}

// ActiveRestaurantResponse is one row of GET /analytics/restaurants/active.
type ActiveRestaurantResponse struct {
	RestaurantID int64 `json:"restaurantId"`
	OrdersCount  int64 `json:"ordersCount"`
}

func ActiveRestaurantResponsesFrom(rows []analytics.ActiveRestaurantRow) []ActiveRestaurantResponse {
	out := make([]ActiveRestaurantResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ActiveRestaurantResponse{
			RestaurantID: row.RestaurantID,
			OrdersCount:  row.OrdersCount,
		})
	}
	return out
}
