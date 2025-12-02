package routes

import (
	"encoding/json"
	"net/http"

	"GoEatsapi/db"
	"GoEatsapi/models"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, name, email FROM login")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Name, &user.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
