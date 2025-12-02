package main

import (
	"fmt"
	"log"
	"net/http"

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

	mux := http.NewServeMux()
	// Setup route to get users

	mux.HandleFunc("/admin_login", routes.AdimnLogin)
	mux.HandleFunc("/admin_get_partners", routes.Get_partners_list)
	mux.HandleFunc("/admin_update_request_status_of_partner", routes.Update_request_status_of_partner)

	mux.HandleFunc("/login", routes.LoginHandler)
	mux.HandleFunc("/users", routes.GetUsers)
	mux.HandleFunc("/signup", routes.SignUp)
	mux.HandleFunc("/Register", routes.RegisterHandler)
	mux.HandleFunc("/verify", routes.VerifyEmailHandler)
	mux.HandleFunc("/Get_user_email_status", routes.GetEmailStatusHandler)
	mux.HandleFunc("/update_partner_details", routes.UpdateDeliveryPartnerHandler)

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	handler := middleware.CORS(mux)

	// Start server
	fmt.Println("🚀 Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		fmt.Println("❌ Server error:", err)
	}
}
