package main

import (
	"fmt"
	"log"
	"net/http"

	"GoEatsapi/config"
	"GoEatsapi/db"
	"GoEatsapi/middleware"
	"GoEatsapi/routes"
	"GoEatsapi/utils"

	"github.com/joho/godotenv"
)

func main() {
	// Connect to MySQL
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()
	utils.InitB2()
	config.InitStripe()

	mux := http.NewServeMux()
	// Admin
	mux.HandleFunc("/admin_login", routes.AdimnLogin)
	mux.HandleFunc("/admin_get_partners", routes.Get_partners_list)
	mux.HandleFunc("/admin_update_request_status_of_partner", routes.Update_request_status_of_partner)

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

	// Partner
	mux.HandleFunc("/login", routes.LoginHandler)
	mux.HandleFunc("/users", routes.GetUsers)
	mux.HandleFunc("/signup", routes.SignUp)
	mux.HandleFunc("/Register", routes.RegisterHandler)
	mux.HandleFunc("/verify", routes.VerifyEmailHandler)
	mux.HandleFunc("/Get_user_email_status", routes.GetEmailStatusHandler)
	mux.HandleFunc("/update_partner_details", routes.UpdateDeliveryPartnerHandler)

	//after login apis
	mux.HandleFunc("/get_partner_details", routes.Get_partner_details)
	mux.HandleFunc("/stripe/create-account", routes.CreateStripeOnboarding)
	mux.HandleFunc("/store_partner_bank_account_details", routes.CreatePartnerBankAccountHandler)
	mux.HandleFunc("/stripe/get-account-details", routes.GetStripe_Account_details_handler)

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	handler := middleware.CORS(mux)

	// Start server
	fmt.Println("🚀 Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		fmt.Println("❌ Server error:", err)
	}
}
