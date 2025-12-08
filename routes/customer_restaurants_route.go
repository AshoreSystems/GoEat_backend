package routes

import (
	"encoding/json"
	"net/http"

	"GoEatsapi/db"
)

// Standard response format
type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Category model
type Category struct {
	ID           int64  `json:"id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// GET all categories API
func GetAllCategories(w http.ResponseWriter, r *http.Request) {

	// Allow only GET request
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	query := `
		SELECT id, category_name, description, status, created_at, updated_at
		FROM categories`

	rows, err := db.DB.Query(query)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch categories")
		return
	}
	defer rows.Close()

	var categories []Category

	for rows.Next() {
		var category Category
		err := rows.Scan(
			&category.ID,
			&category.CategoryName,
			&category.Description,
			&category.Status,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			sendErrorResponse(w, "Failed to parse category data")
			return
		}
		categories = append(categories, category)
	}

	response := Response{
		Status:  true,
		Message: "Categories fetched successfully",
		Data:    categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type CategoryRequest struct {
	CategoryID int `json:"category_id"`
}
type MenuItem struct {
	MenuItemID  int     `json:"menu_item_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image_url"`
}

type RestaurantData struct {
	RestaurantID        int        `json:"restaurant_id"`
	RestaurantName      string     `json:"restaurant_name"`
	BusinessDescription string     `json:"business_description"`
	CoverImages         string     `json:"cover_image"`
	Rating              float64    `json:"rating"`
	Items               []MenuItem `json:"items"`
}

type APIResponse struct {
	Status  bool             `json:"status"`
	Message string           `json:"message"`
	Data    []RestaurantData `json:"data"`
}

func GetRestaurantsByCategory(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CategoryID == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Updated Query
	query := `
	SELECT 
		r.id AS restaurant_id,
		r.restaurant_name,
		r.business_description,
		r.cover_image,
		r.rating,
		mi.id AS menu_item_id,
		mi.item_name,
		mi.price
	FROM restaurants r
	JOIN restaurant_menu_items rmi ON r.id = rmi.restaurant_id
	JOIN menu_items mi ON rmi.menu_item_id = mi.id
	WHERE mi.category_id = ?
	ORDER BY r.restaurant_name, mi.item_name;
	`

	rows, err := db.DB.Query(query, req.CategoryID)
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	restaurantsMap := make(map[int]*RestaurantData)

	for rows.Next() {
		var rID int
		var rName, businessDesc, CoverImages string
		var rating float64
		var item MenuItem

		err := rows.Scan(&rID, &rName, &businessDesc, &CoverImages, &rating, &item.MenuItemID, &item.Name, &item.Price)
		if err != nil {
			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if _, exists := restaurantsMap[rID]; !exists {
			restaurantsMap[rID] = &RestaurantData{
				RestaurantID:        rID,
				RestaurantName:      rName,
				BusinessDescription: businessDesc,
				CoverImages:         CoverImages,
				Rating:              rating,
				Items:               []MenuItem{},
			}
		}

		restaurantsMap[rID].Items = append(restaurantsMap[rID].Items, item)
	}

	restaurantsList := []RestaurantData{}
	for _, v := range restaurantsMap {
		restaurantsList = append(restaurantsList, *v)
	}

	response := APIResponse{
		Status:  true,
		Message: "Restaurant list fetched successfully",
		Data:    restaurantsList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type MenuRequest struct {
	RestaurantID int `json:"restaurant_id"`
}

// type MenuItem struct {
// 	MenuItemID  int     `json:"menu_item_id"`
// 	Name        string  `json:"name"`
// 	Description string  `json:"description"`
// 	Price       float64 `json:"price"`
// 	Image       string  `json:"image"`
// }

type CategoryMenu struct {
	CategoryID   int        `json:"category_id"`
	CategoryName string     `json:"category_name"`
	Items        []MenuItem `json:"items"`
}

type MenuItemResponse struct {
	Status  bool           `json:"status"`
	Message string         `json:"message"`
	Data    []CategoryMenu `json:"data"`
}

func GetRestaurantMenu(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req MenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RestaurantID == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
	SELECT 
		c.id AS category_id,
		c.category_name,
		mi.id AS menu_item_id,
		mi.item_name,
		mi.description,
		mi.price,
		mi.image_url
	FROM categories c
	JOIN menu_items mi ON mi.category_id = c.id
	JOIN restaurant_menu_items rmi ON rmi.menu_item_id = mi.id
	WHERE rmi.restaurant_id = ?
	ORDER BY c.category_name, mi.item_name;
	`

	rows, err := db.DB.Query(query, req.RestaurantID)
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categoryMap := make(map[int]*CategoryMenu)

	for rows.Next() {
		var categoryID int
		var categoryName, desc, image_url string
		var item MenuItem

		err := rows.Scan(&categoryID, &categoryName, &item.MenuItemID, &item.Name, &desc, &item.Price, &image_url)
		if err != nil {
			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		item.Description = desc
		item.Image = image_url

		if _, exists := categoryMap[categoryID]; !exists {
			categoryMap[categoryID] = &CategoryMenu{
				CategoryID:   categoryID,
				CategoryName: categoryName,
				Items:        []MenuItem{},
			}
		}

		categoryMap[categoryID].Items = append(categoryMap[categoryID].Items, item)
	}

	// Convert map to list
	menuList := []CategoryMenu{}
	for _, v := range categoryMap {
		menuList = append(menuList, *v)
	}

	response := MenuItemResponse{
		Status:  true,
		Message: "Restaurant menu list fetched successfully",
		Data:    menuList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type MenuDetailRequest struct {
	MenuItemID int `json:"menu_item_id"`
}

type MenuDetails struct {
	MenuItemID     int     `json:"menu_item_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Price          float64 `json:"price"`
	Image          string  `json:"image_url"`
	Status         string  `json:"status"`
	CategoryID     int     `json:"category_id"`
	CategoryName   string  `json:"category_name"`
	RestaurantID   int     `json:"restaurant_id"`
	RestaurantName string  `json:"restaurant_name"`
}

type MenuDetailResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    MenuDetails `json:"data"`
}

func GetMenuItemDetails(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		//http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		sendErrorResponse(w, "Invalid request method")
		return
	}

	// Parse request body
	var req MenuDetailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MenuItemID == 0 {
		//http.Error(w, "Invalid request body", http.StatusBadRequest)
		sendErrorResponse(w, "Invalid request body")
		return
	}

	query := `
	SELECT 
		mi.id AS menu_item_id,
		mi.item_name,
		mi.description,
		mi.price,
		mi.image_url,
		mi.status,
		c.id AS category_id,
		c.category_name,
		r.id AS restaurant_id,
		r.restaurant_name
	FROM menu_items mi
	JOIN categories c ON mi.category_id = c.id
	JOIN restaurant_menu_items rmi ON rmi.menu_item_id = mi.id
	JOIN restaurants r ON r.id = rmi.restaurant_id
	WHERE mi.id = ?
	LIMIT 1;
	`

	var details MenuDetails
	err := db.DB.QueryRow(query, req.MenuItemID).Scan(
		&details.MenuItemID,
		&details.Name,
		&details.Description,
		&details.Price,
		&details.Image,
		&details.Status,
		&details.CategoryID,
		&details.CategoryName,
		&details.RestaurantID,
		&details.RestaurantName,
	)

	if err != nil {
		//	http.Error(w, "Menu item not found", http.StatusNotFound)
		sendErrorResponse(w, "Menu item not found")
		return
	}

	response := MenuDetailResponse{
		Status:  true,
		Message: "Menu item details fetched successfully",
		Data:    details,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
