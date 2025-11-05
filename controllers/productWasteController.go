package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// GetTotalWaste - Get total waste weight
func GetTotalWaste(c *fiber.Ctx) error {
	var totalWeight struct {
		Total float64 `json:"total"`
	}

	// Calculate total waste weight
	if err := configs.DB.Model(&models.ProductWaste{}).
		Select("COALESCE(SUM(weight), 0) as total").
		Scan(&totalWeight).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to calculate total waste", nil, nil)
	}

	response := map[string]any{
		"total_weight": totalWeight.Total,
		"unit":         "kilogram",
	}

	return helpers.Response(c, 200, "Success", "Total waste retrieved successfully", response, nil)
}

// GetProductWasteList - Get all product waste with filtering
func GetProductWasteList(c *fiber.Ctx) error {
	var query struct {
		Search   string `query:"search"`
		Category string `query:"category"`
		Page     int    `query:"page"`
		Limit    int    `query:"limit"`
	}

	if err := c.QueryParser(&query); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid query parameters", nil, nil)
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	var products []models.ProductWaste
	dbQuery := configs.DB

	// Apply search filter
	if query.Search != "" {
		search := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("waste_type LIKE ?", search)
	}

	// Apply category filter
	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}

	var total int64
	dbQuery.Model(&models.ProductWaste{}).Count(&total)

	// Get data dengan pagination
	if err := dbQuery.Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&products).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch product waste", nil, nil)
	}

	// Format response
	var formattedProducts []map[string]any
	for i, product := range products {
		formattedProducts = append(formattedProducts, map[string]any{
			"no":           i + 1 + offset,
			"id":           product.Id,
			"waste_type":   product.WasteType,
			"image":        product.Image,
			"unit":         product.Unit,
			"price":        product.Price,
			"price_format": FormatCurrency(product.Price),
			"category":     product.Category,
			// "weight":     product.Weight,
		})
	}

	response := map[string]any{
		"products": formattedProducts,
		"page":     query.Page,
		"limit":    query.Limit,
		"total":    total,
	}

	return helpers.Response(c, 200, "Success", "Product waste retrieved successfully", response, nil)
}

// CreateProductWaste - Create new product waste
func CreateProductWaste(c *fiber.Ctx) error {
	var body struct {
		WasteType string `json:"waste_type"`
		Image     string `json:"image"`
		Unit      string `json:"unit"`
		Price     int    `json:"price"`
		Category  string `json:"category"`
		// Weight    float64 `json:"weight"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Validate required fields
	if body.WasteType == "" || body.Unit == "" || body.Category == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Waste type, unit, and category are required", nil, nil)
	}

	// Validate category
	if body.Category != "organik" && body.Category != "anorganik" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Category must be 'organic' or 'inorganic'", nil, nil)
	}

	product := models.ProductWaste{
		WasteType: body.WasteType,
		Image:     body.Image,
		Unit:      body.Unit,
		Price:     body.Price,
		Category:  body.Category,
		// Weight:    body.Weight,
	}

	if err := configs.DB.Create(&product).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to create product waste", nil, nil)
	}

	return helpers.Response(c, 201, "Success", "Product waste created successfully", product, nil)
}

// UpdateProductWaste - Update product waste
func UpdateProductWaste(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid product ID", nil, nil)
	}

	var body struct {
		WasteType string `json:"waste_type"`
		Image     string `json:"image"`
		Unit      string `json:"unit"`
		Price     int    `json:"price"`
		Category  string `json:"category"`
		// Weight    float64 `json:"weight"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	var product models.ProductWaste
	if err := configs.DB.First(&product, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Product waste not found", nil, nil)
	}

	// Update fields
	if body.WasteType != "" {
		product.WasteType = body.WasteType
	}
	if body.Image != "" {
		product.Image = body.Image
	}
	if body.Unit != "" {
		product.Unit = body.Unit
	}
	if body.Price != 0 {
		product.Price = body.Price
	}
	if body.Category != "" {
		product.Category = body.Category
	}
	// if body.Weight != 0 {
	// 	product.Weight = body.Weight
	// }

	if err := configs.DB.Save(&product).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update product waste", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Product waste updated successfully", product, nil)
}

// DeleteProductWaste - Delete product waste
func DeleteProductWaste(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid product ID", nil, nil)
	}

	var product models.ProductWaste
	if err := configs.DB.First(&product, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Product waste not found", nil, nil)
	}

	if err := configs.DB.Delete(&product).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to delete product waste", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Product waste deleted successfully", nil, nil)
}

// Helper function untuk format currency (simple version)
func FormatCurrency(amount int) string {
	return "Rp" + strconv.Itoa(amount)
}
