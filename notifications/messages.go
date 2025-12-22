package notifications

import "fmt"

// Order Delivered
func OrderDelivered(orderNumber string) (string, string) {
	title := "Order Delivered 🎉"
	body := fmt.Sprintf(
		"Your order %s has been delivered successfully. Thank you for choosing GoEats!",
		orderNumber,
	)
	return title, body
}

// Order Picked Up
func OrderPickedUp(orderNumber string) (string, string) {
	title := "Order Picked Up 🚴"
	body := fmt.Sprintf(
		"Good news! Your order %s has been picked up and is on the way.",
		orderNumber,
	)
	return title, body
}

// Order Cancelled
func OrderCancelled(orderNumber string) (string, string) {
	title := "Order Cancelled ❌"
	body := fmt.Sprintf(
		"Your order %s has been cancelled. Any refunds will be processed shortly.",
		orderNumber,
	)
	return title, body
}
