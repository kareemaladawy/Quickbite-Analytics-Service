package entity

// ActiveRestaurant is the $group-stage result shape for the active-restaurants
// query — not a stored document, just the aggregation pipeline's output
// decoded straight off agg_restaurant_day.
type ActiveRestaurant struct {
	RestaurantID int64 `bson:"_id"`
	OrdersCount  int64 `bson:"orders_count"`
}
