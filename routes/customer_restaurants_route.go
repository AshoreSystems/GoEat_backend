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
	CategoryID int     `json:"category_id"`
	CustomerID uint64  `json:"customer_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	RadiusKM   float64 `json:"radius_km"`
}
type MenuItem struct {
	MenuItemID  int     `json:"menu_item_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image_url"`
	IsVeg       bool    `json:"is_veg"`
}

type RestaurantData struct {
	RestaurantID        int        `json:"restaurant_id"`
	RestaurantName      string     `json:"restaurant_name"`
	BusinessDescription string     `json:"business_description"`
	CoverImages         string     `json:"cover_image"`
	Rating              float64    `json:"rating"`
	Wishlist            bool       `json:"wishlist"`
	DistanceKM          float64    `json:"distance_km"`
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

	query := `
	SELECT 
	r.id AS restaurant_id,
	r.restaurant_name,
	COALESCE(r.business_description, '') AS business_description,
	COALESCE(r.cover_image, '') AS cover_image,
	r.rating,
	mi.id AS menu_item_id,
	mi.item_name,
	mi.price,
	(
		6371 * ACOS(
			COS(RADIANS(?)) *
			COS(RADIANS(r.latitude)) *
			COS(RADIANS(r.longitude) - RADIANS(?)) +
			SIN(RADIANS(?)) *
			SIN(RADIANS(r.latitude))
		)
	) AS distance
FROM restaurants r
JOIN restaurant_menu_items rmi ON r.id = rmi.restaurant_id
JOIN menu_items mi ON rmi.menu_item_id = mi.id
WHERE 
	mi.category_id = ?
	AND r.status = 'approved'
HAVING distance <= ?
ORDER BY distance, r.restaurant_name, mi.item_name;`

	//rows, err := db.DB.Query(query, req.CategoryID)

	radius := req.RadiusKM
	if radius == 0 {
		radius = 10 // default 10 km
	}

	rows, err := db.DB.Query(
		query,
		req.Latitude,
		req.Longitude,
		req.Latitude,
		req.CategoryID,
		radius,
	)
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
		var distance float64
		err := rows.Scan(&rID, &rName, &businessDesc, &CoverImages, &rating, &item.MenuItemID, &item.Name, &item.Price, &distance)
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
				Wishlist:            IsWishlist(req.CustomerID, rID),
				DistanceKM:          distance,
				Items:               []MenuItem{},
			}

		}

		// Add menu items
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

func IsWishlist(customerID uint64, restaurantID int) bool {
	var count int
	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM tbl_customer_wishlist WHERE customer_id = ? AND restaurant_id = ?",
		customerID, restaurantID,
	).Scan(&count)

	if err != nil {
		return false
	}
	return count > 0
}

type MenuRequest struct {
	RestaurantID int `json:"restaurant_id"`
	CategoryID   int `json:"category_id"`
}

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
	mi.image_url,
	mi.is_veg
FROM categories c
JOIN menu_items mi ON mi.category_id = c.id
JOIN restaurant_menu_items rmi ON rmi.menu_item_id = mi.id
WHERE 
	rmi.restaurant_id = ?
	AND mi.status = 'active'
	AND (? = 0 OR c.id = ?)
ORDER BY c.category_name, mi.item_name;

	`

	//rows, err := db.DB.Query(query, req.RestaurantID)
	rows, err := db.DB.Query(
		query,
		req.RestaurantID,
		req.CategoryID,
		req.CategoryID,
	)
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categoryMap := make(map[int]*CategoryMenu)

	for rows.Next() {
		var categoryID int
		var categoryName string
		var item MenuItem

		err := rows.Scan(
			&categoryID,
			&categoryName,
			&item.MenuItemID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Image,
			&item.IsVeg,
		)
		if err != nil {
			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}

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

type NearbyMenuRequest struct {
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Radius     float64 `json:"radius"`
	CategoryID int     `json:"category_id"` // new optional filter
}
type RestaurentMenuItem struct {
	RestaurantID   int     `json:"restaurant_id"`
	RestaurantName string  `json:"restaurant_name"`
	CategoryID     int     `json:"category_id"`
	CategoryName   string  `json:"category_name"`
	MenuItemID     int     `json:"menu_item_id"`
	ItemName       string  `json:"item_name"`
	Description    string  `json:"description"`
	Price          float64 `json:"price"`
	ImageURL       string  `json:"image_url"`
	IsVeg          int     `json:"is_veg"`
	Distance       float64 `json:"distance"`
}

func GetNearbyRestaurantMenu(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req NearbyMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	//db := GetDB()

	baseQuery := `
	SELECT
		r.id AS restaurant_id,
		r.restaurant_name AS restaurant_name,
		c.id AS category_id,
		c.category_name,
		mi.id AS menu_item_id,
		mi.item_name,
		mi.description,
		mi.price,
		mi.image_url,
		mi.is_veg,
		r.distance
	FROM (
		SELECT 
			id, restaurant_name,
			(6371 * acos(
				cos(radians(?)) * cos(radians(latitude)) *
				cos(radians(longitude) - radians(?)) +
				sin(radians(?)) * sin(radians(latitude))
			)) AS distance
		FROM restaurants
		HAVING distance <= ?
	) r
	JOIN restaurant_menu_items rmi ON rmi.restaurant_id = r.id
	JOIN menu_items mi ON mi.id = rmi.menu_item_id
	JOIN categories c ON c.id = mi.category_id
	WHERE mi.status = 'active'
	`

	// Add category filter condition dynamically
	args := []interface{}{req.Latitude, req.Longitude, req.Latitude, req.Radius}

	if req.CategoryID > 0 {
		baseQuery += " AND c.id = ? "
		args = append(args, req.CategoryID)
	}

	baseQuery += " ORDER BY r.distance, c.category_name, mi.item_name;"

	rows, err := db.DB.Query(baseQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	menuList := []RestaurentMenuItem{}
	for rows.Next() {
		var m RestaurentMenuItem
		err := rows.Scan(&m.RestaurantID, &m.RestaurantName, &m.CategoryID, &m.CategoryName,
			&m.MenuItemID, &m.ItemName, &m.Description, &m.Price, &m.ImageURL, &m.IsVeg, &m.Distance)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		menuList = append(menuList, m)
	}

	// ------------------ GROUP BY CATEGORY ------------------
	items := []map[string]interface{}{}

	for _, item := range menuList {
		items = append(items, map[string]interface{}{
			"restaurant_id":   item.RestaurantID,
			"restaurant_name": item.RestaurantName,
			"menu_item_id":    item.MenuItemID,
			"item_name":       item.ItemName,
			"description":     item.Description,
			"price":           item.Price,
			"image_url":       item.ImageURL,
			"is_veg":          item.IsVeg,
			"distance":        item.Distance,
		})
	}

	response := MenuResponse{
		Status: true,
		Items:  items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type MenuResponse struct {
	Status bool        `json:"status"`
	Items  interface{} `json:"items"`
}

func AddToWishlist(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req struct {
		CustomerID   int    `json:"customer_id"`
		RestaurantID uint64 `json:"restaurant_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// STEP 1: Check if already in wishlist
	var count int
	checkQuery := `
        SELECT COUNT(*) FROM tbl_customer_wishlist
        WHERE customer_id = ? AND restaurant_id = ?
    `
	err = db.DB.QueryRow(checkQuery, req.CustomerID, req.RestaurantID).Scan(&count)
	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error())
		return
	}

	if count > 0 {
		// Already exists
		resp := map[string]interface{}{
			"status":  false,
			"message": "Restaurant already added to wishlist",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// STEP 2: Insert wishlist entry
	insertQuery := `
        INSERT INTO tbl_customer_wishlist (customer_id, restaurant_id)
        VALUES (?, ?)
    `
	_, err = db.DB.Exec(insertQuery, req.CustomerID, req.RestaurantID)
	if err != nil {
		sendErrorResponse(w, "Failed to add to wishlist: "+err.Error())
		return
	}

	// Success Response
	resp := map[string]interface{}{
		"status":  true,
		"message": "Added to wishlist",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func RemoveFromWishlist(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req struct {
		CustomerID   int    `json:"customer_id"`
		RestaurantID uint64 `json:"restaurant_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	query := `
        DELETE FROM tbl_customer_wishlist
        WHERE customer_id = ? AND restaurant_id = ?
    `
	_, err = db.DB.Exec(query, req.CustomerID, req.RestaurantID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove from wishlist: "+err.Error())
		return
	}

	resp := map[string]interface{}{
		"status":  true,
		"message": "Removed from wishlist",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetWishlist(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req struct {
		CustomerID uint64 `json:"customer_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.CustomerID == 0 {
		sendErrorResponse(w, "Invalid or missing customer_id")
		return
	}

	query := `
        SELECT 
            r.id,
            r.restaurant_name,
            r.business_description,
            r.rating,
            r.cover_image
        FROM tbl_customer_wishlist w
        JOIN restaurants r ON w.restaurant_id = r.id
        WHERE w.customer_id = ?
    `

	rows, err := db.DB.Query(query, req.CustomerID)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch wishlist: "+err.Error())
		return
	}
	defer rows.Close()
	type WishlistItem struct {
		ID             uint64  `json:"id"`
		RestaurantID   uint64  `json:"restaurant_id"`
		RestaurantName string  `json:"restaurant_name"`
		BusinessDesc   string  `json:"business_description"`
		Rating         float64 `json:"rating"`
		CoverImage     *string `json:"cover_image"`
		Wishlist       bool    `json:"wishlist"`
	}

	var list []WishlistItem

	for rows.Next() {
		var item WishlistItem

		err := rows.Scan(
			&item.ID,
			&item.RestaurantName,
			&item.BusinessDesc,
			&item.Rating,
			&item.CoverImage,
		)

		if err == nil {
			item.RestaurantID = item.ID // assign explicitly
			item.Wishlist = true
			list = append(list, item)
		}
	}

	// ---- IF NO DATA FOUND ----
	if len(list) == 0 {
		resp := map[string]interface{}{
			"status":  false,
			"message": "No wishlist found",
			"data":    []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// ---- SUCCESS RESPONSE ----
	resp := map[string]interface{}{
		"status": true,
		"data":   list,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type RestaurantMenuRequest struct {
	RestaurantID uint64 `json:"restaurant_id"`
}
type RestMenuItem struct {
	MenuItemID      uint64  `json:"menu_item_id"`
	ItemName        string  `json:"item_name"`
	Description     string  `json:"description"`
	CategoryID      uint64  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	Price           float64 `json:"price"`
	IsVeg           int     `json:"is_veg"`
	IsAvailable     int     `json:"is_available"`
	PreparationTime *int    `json:"preparation_time"`
	ImageURL        *string `json:"image_url"`
}

type Restaurant struct {
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	IsOpen    int     `json:"is_open"`
	OpenTime  *string `json:"open_time"`
	CloseTime *string `json:"close_time"`
	Rating    float32 `json:"rating"`
}

type RestaurantMenuResponse struct {
	Restaurant Restaurant     `json:"restaurant"`
	Items      []RestMenuItem `json:"items"`
}

func GetAllRestaurantMenu(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req RestaurantMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RestaurantID == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(`
		SELECT
			r.id,
			r.restaurant_name,
			r.is_open,
			r.open_time,
			r.close_time,
			r.rating,

			c.id,
			c.category_name,

			mi.id,
			mi.item_name,
			mi.description,
			COALESCE(rmi.price, mi.price),
			mi.is_veg,
			rmi.is_available,
			COALESCE(rmi.preparation_time, mi.preparation_time),
			mi.image_url
		FROM restaurants r
		JOIN restaurant_menu_items rmi ON r.id = rmi.restaurant_id
		JOIN menu_items mi ON rmi.menu_item_id = mi.id
		JOIN categories c ON mi.category_id = c.id
		WHERE r.id = ?
		  AND rmi.status = 'active'
		  AND mi.status = 'active'
		ORDER BY mi.item_name
	`, req.RestaurantID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response RestaurantMenuResponse
	var restaurantSet bool

	for rows.Next() {
		var item RestMenuItem
		var restaurant Restaurant

		err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.IsOpen,
			&restaurant.OpenTime,
			&restaurant.CloseTime,
			&restaurant.Rating,

			&item.CategoryID,
			&item.CategoryName,

			&item.MenuItemID,
			&item.ItemName,
			&item.Description,
			&item.Price,
			&item.IsVeg,
			&item.IsAvailable,
			&item.PreparationTime,
			&item.ImageURL,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !restaurantSet {
			response.Restaurant = restaurant
			restaurantSet = true
		}

		response.Items = append(response.Items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
