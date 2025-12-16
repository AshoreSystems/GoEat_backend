package main

import (
	"fmt"
	"log"
	"net/http"

	"GoEatsapi/config"
	"GoEatsapi/db"
	"GoEatsapi/middleware"
	"GoEatsapi/routes"
	resto "GoEatsapi/routes/Resto"
	Admin "GoEatsapi/routes/admin"
	DeliveryPartner "GoEatsapi/routes/delivery_partner"
	"GoEatsapi/utils"

	"github.com/joho/godotenv"
)

var (
	appLog   *log.Logger
	errorLog *log.Logger
)

func main() {
	// Connect to MySQL
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()
	config.InitStripe()

	utils.InitLogger()

	utils.InfoLog.Println("Server started")
	utils.ErrorLog.Println("error found")

	mux := http.NewServeMux()
	// Admin
	mux.HandleFunc("/admin_login", Admin.AdimnLogin)
	mux.HandleFunc("/admin_get_partners", Admin.Get_partners_list)
	mux.HandleFunc("/admin_update_request_status_of_partner", Admin.Update_request_status_of_partner)
	mux.HandleFunc("/admin_get_restaurants", Admin.Get_restaurants_list)
	mux.HandleFunc("/admin_update_request_status_of_restaurant", Admin.Update_request_status_of_restaurant)

	//resto
	mux.HandleFunc("/api/resto-signin", resto.RestoLogin)
	mux.HandleFunc("/api/resto-signup", resto.RestoRegister)
	mux.HandleFunc("/api/resto-RestoDetails", resto.RestoCheckDetails)
	mux.HandleFunc("/api/resto-orders", resto.GetRestoOrders)
	mux.HandleFunc("/api/resto-orders/accept", resto.UpdateOrderStatus)
	mux.HandleFunc("/api/resto-stripe/get-account-details", resto.GetStripe_Account_details_handler)
	mux.HandleFunc("/api/resto-stripe/create-account", resto.CreateStripeRestaurantOnboarding)
	mux.HandleFunc("/api/resto-Categories", resto.GetRestoCategories)
	mux.HandleFunc("/api/add-menu-item", resto.AddMenuItem)
	mux.HandleFunc("/api/update-menu-item", resto.UpdateMenuItem)
	mux.HandleFunc("/api/resto-menu", resto.GetMenuByRestaurant)
	mux.HandleFunc("/api/resto-menu-disable", resto.DisableMenuItem)
	mux.HandleFunc("/api/update-resto-address", resto.UpdateRestaurantAddress)
	mux.HandleFunc("/api/update-resto-time", resto.UpdateRestaurantTime)

	//Customer
	mux.HandleFunc("/signup-customer", routes.SingUp_Customer)
	mux.HandleFunc("/login-customer", routes.LoginCustomer)
	mux.HandleFunc("/verify-customer", routes.CustomerVerifyOTP)
	mux.HandleFunc("/resend-otp-customer", routes.CustomerResendOTP)
	mux.HandleFunc("/categories", middleware.AuthMiddleware(routes.GetAllCategories))
	mux.HandleFunc("/restaurants-by-category", middleware.AuthMiddleware(routes.GetRestaurantsByCategory))
	mux.HandleFunc("/restaurants-menu", middleware.AuthMiddleware(routes.GetRestaurantMenu))
	mux.HandleFunc("/menu-details", middleware.AuthMiddleware(routes.GetMenuItemDetails))
	mux.HandleFunc("/customer-details", middleware.AuthMiddleware(routes.GetCustomerDetails))
	mux.HandleFunc("/customer-add-delivery-address", middleware.AuthMiddleware(routes.AddCustomerAddress))
	mux.HandleFunc("/customer-delivery-address-list", middleware.AuthMiddleware(routes.GetCustomerAddresses))
	mux.HandleFunc("/delete-customer-address", middleware.AuthMiddleware(routes.DeleteCustomerAddress))
	mux.HandleFunc("/near-by-restaurant-menu", middleware.AuthMiddleware(routes.GetNearbyRestaurantMenu))
	mux.HandleFunc("/customer-update-profile", middleware.AuthMiddleware(routes.UpdateCustomerProfile))
	mux.HandleFunc("/customer-place-order", middleware.AuthMiddleware(routes.PlaceOrder))
	mux.HandleFunc("/payment/create-intent", middleware.AuthMiddleware(routes.Create_payment_intent))
	mux.HandleFunc("/get-default-address", middleware.AuthMiddleware(routes.GetDefaultAddress))
	mux.HandleFunc("/update-default-address", middleware.AuthMiddleware(routes.UpdateDefaultAddress))
	mux.HandleFunc("/my-orders", middleware.AuthMiddleware(routes.GetCustomerOrders))
	mux.HandleFunc("/cancel-customer-order", middleware.AuthMiddleware(routes.CancelCustomerOrder))
	mux.HandleFunc("/order-rating-review", middleware.AuthMiddleware(routes.CreateRatingReview))
	mux.HandleFunc("/add-wishlist", middleware.AuthMiddleware(routes.AddToWishlist))
	mux.HandleFunc("/remove-wishlist", middleware.AuthMiddleware(routes.RemoveFromWishlist))
	mux.HandleFunc("/get-wishlist", middleware.AuthMiddleware(routes.GetWishlist))
	mux.HandleFunc("/contact-us", middleware.AuthMiddleware(routes.CreateContactUs))
	// Partner
	mux.HandleFunc("/login", routes.LoginHandler)
	mux.HandleFunc("/users", routes.GetUsers)
	mux.HandleFunc("/signup", DeliveryPartner.SignUp)
	mux.HandleFunc("/Register", routes.RegisterHandler)
	mux.HandleFunc("/verify", routes.VerifyEmailHandler)
	mux.HandleFunc("/Get_user_email_status", routes.GetEmailStatusHandler)
	mux.HandleFunc("/update_partner_details", DeliveryPartner.UpdateDeliveryPartnerHandler)
	mux.HandleFunc("/api/partner/orders_by_status", DeliveryPartner.GetPartnerOrder)
	mux.HandleFunc("/api/partner/get-dashboard-graph", DeliveryPartner.GetOrderGraph)

	//after login apis
	mux.HandleFunc("/get_partner_details", DeliveryPartner.Get_partner_details)
	mux.HandleFunc("/stripe/create-account", DeliveryPartner.CreateStripeOnboarding)
	mux.HandleFunc("/store_partner_bank_account_details", DeliveryPartner.CreatePartnerBankAccountHandler)
	mux.HandleFunc("/stripe/get-account-details", DeliveryPartner.GetStripe_Account_details_handler)

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	handler := middleware.CORS(mux)

	// Start server
	fmt.Println("🚀 Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		// if err := http.ListenAndServe(":8013", handler); err != nil {
		fmt.Println("❌ Server error:", err)
	}
}
