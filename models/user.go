package models

import (
	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zipcode     string `json:"zipcode"`
	Country     string `json:"country"`
	Gender      string `json:"gender"`
	DateOfBirth string `json:"date_of_birth"`
	UserType    string `json:"user_type"`
	ProfilePic  string `json:"profile_pic"`
	IDNumber    string `json:"id_number"`
	IDDocFront  string `json:"id_doc_front"`
	IDDocBack   string `json:"id_doc_back"`
}

type Claims struct {
	LoginID int    `json:"login_id"`
	Email   string `json:"email"`
	jwt.RegisteredClaims
}
