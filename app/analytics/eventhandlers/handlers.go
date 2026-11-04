// Package eventhandlers maps event type -> service method, decoding each
// event's JSON payload into the input type the service expects. This is the
// one place order-service's wire payload shape and this service's internal
// types meet.
package eventhandlers

import (
	"encoding/json"
	"fmt"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/app/analytics/service"
	"quickbite/analytics-service/lib/coreevents"
)

// orderPlacedItemPayload mirrors order-service's OrderOutboxItemPayload
// (app/order/types.ts), one entry of OrderPlacedOutboxPayload.items.
type orderPlacedItemPayload struct {
	ProductID int64 `json:"productId"`
	Quantity  int64 `json:"quantity"`
	LineTotal int64 `json:"lineTotal"`
}

// orderPlacedPayload mirrors the fields of order-service's
// OrderPlacedOutboxPayload (app/order/types.ts) this handler needs — not
// the full payload (customerId, status, etc. are unused by the aggregates
// this handler feeds).
type orderPlacedPayload struct {
	OrderID      int64                    `json:"orderId"`
	RestaurantID int64                    `json:"restaurantId"`
	BranchID     int64                    `json:"branchId"`
	Total        int64                    `json:"total"`
	Currency     string                   `json:"currency"`
	Items        []orderPlacedItemPayload `json:"items"`
	PlacedAt     string                   `json:"placedAt"`
}

// orderLifecyclePayload mirrors the fields of order-service's
// OrderOutboxPayload (app/order/types.ts) this module's order.rejected/
// order.delivered handlers need — not the full snapshot (status,
// commission, cancellationReason, etc. are unused here).
type orderLifecyclePayload struct {
	OrderID      int64  `json:"id"`
	RestaurantID int64  `json:"restaurantId"`
	BranchID     int64  `json:"branchId"`
	CreatedAt    string `json:"createdAt"`
	DeliveredAt  string `json:"deliveredAt"`
}

// paymentCompletedPayload mirrors order-service's PaymentOutboxPayload
// (app/payment/types.ts), trimmed to what agg_platform_day needs.
type paymentCompletedPayload struct {
	TransactionID int64  `json:"transactionId"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"createdAt"`
}

// Register wires every handler this service implements into the consumer.
func Register(consumer *coreevents.Consumer, svc *service.AnalyticsService) {
	consumer.RegisterHandler(analytics.OrderEventTypePlaced, handleOrderPlaced(svc))
	consumer.RegisterHandler(analytics.OrderEventTypeRejected, handleOrderRejected(svc))
	consumer.RegisterHandler(analytics.OrderEventTypeDelivered, handleOrderDelivered(svc))
	consumer.RegisterHandler(analytics.PaymentEventTypeCompleted, handlePaymentCompleted(svc))
}

func handleOrderPlaced(svc *service.AnalyticsService) coreevents.EventHandler {
	return func(payload json.RawMessage) error {
		var decoded orderPlacedPayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("eventhandlers: decode order.placed payload: %w", err)
		}

		items := make([]analytics.OrderPlacedItemInput, 0, len(decoded.Items))
		for _, item := range decoded.Items {
			items = append(items, analytics.OrderPlacedItemInput{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				LineTotal: item.LineTotal,
			})
		}

		return svc.OnOrderPlaced(analytics.OnOrderPlacedInput{
			OrderID:      decoded.OrderID,
			RestaurantID: decoded.RestaurantID,
			BranchID:     decoded.BranchID,
			Total:        decoded.Total,
			Currency:     decoded.Currency,
			Items:        items,
			PlacedAt:     decoded.PlacedAt,
		})
	}
}

func handleOrderRejected(svc *service.AnalyticsService) coreevents.EventHandler {
	return func(payload json.RawMessage) error {
		var decoded orderLifecyclePayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("eventhandlers: decode order.rejected payload: %w", err)
		}

		return svc.OnOrderRejected(analytics.OnOrderRejectedInput{
			OrderID:      decoded.OrderID,
			RestaurantID: decoded.RestaurantID,
			BranchID:     decoded.BranchID,
			CreatedAt:    decoded.CreatedAt,
		})
	}
}

func handleOrderDelivered(svc *service.AnalyticsService) coreevents.EventHandler {
	return func(payload json.RawMessage) error {
		var decoded orderLifecyclePayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("eventhandlers: decode order.delivered payload: %w", err)
		}

		return svc.OnOrderDelivered(analytics.OnOrderDeliveredInput{
			OrderID:      decoded.OrderID,
			RestaurantID: decoded.RestaurantID,
			BranchID:     decoded.BranchID,
			CreatedAt:    decoded.CreatedAt,
			DeliveredAt:  decoded.DeliveredAt,
		})
	}
}

func handlePaymentCompleted(svc *service.AnalyticsService) coreevents.EventHandler {
	return func(payload json.RawMessage) error {
		var decoded paymentCompletedPayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("eventhandlers: decode payment.completed payload: %w", err)
		}

		return svc.OnPaymentCompleted(analytics.OnPaymentCompletedInput{
			TransactionID: decoded.TransactionID,
			Amount:        decoded.Amount,
			Currency:      decoded.Currency,
			CreatedAt:     decoded.CreatedAt,
		})
	}
}
