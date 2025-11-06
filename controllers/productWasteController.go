package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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

// CreateProductWaste - Create new product waste dengan upload foto ke AWS S3
func CreateProductWaste(c *fiber.Ctx) error {
	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form
	wasteType := getFirstValue(form.Value["waste_type"])
	unit := getFirstValue(form.Value["unit"])
	category := getFirstValue(form.Value["category"])
	priceStr := getFirstValue(form.Value["price"])

	// Validate required fields
	if wasteType == "" || unit == "" || category == "" || priceStr == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Waste type, unit, category, and price are required", nil, nil)
	}

	// Validate category
	if category != "organik" && category != "anorganik" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Category must be 'organik' or 'anorganik'", nil, nil)
	}

	// Convert price to int
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid price format", nil, nil)
	}

	// Initialize product waste dengan data dasar
	product := models.ProductWaste{
		WasteType: wasteType,
		Unit:      unit,
		Price:     price,
		Category:  category,
		Image:     "", // Default empty, akan diupdate jika ada upload foto
	}

	// Handle file upload jika ada
	files := form.File["image"]
	if len(files) > 0 {
		file := files[0]

		fmt.Printf("File received: %s, Size: %d bytes\n", file.Filename, file.Size)

		// Validate file size (max 2MB)
		if file.Size > 2<<20 {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "File size too large (max 2MB)", nil, nil)
		}

		// Validate file type
		allowedTypes := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedTypes[ext] {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
		}

		fmt.Printf("File type validated: %s\n", ext)

		// Initialize S3 service
		s3Service := helpers.NewS3Service()
		fmt.Printf("S3 service initialized\n")

		// Upload to S3 - menggunakan ID product 0 (sementara)
		fmt.Printf("Starting S3 upload for product waste image...\n")
		imageURL, err := s3Service.UploadFile(file, 0, "product-waste")
		if err != nil {
			fmt.Printf("S3 upload failed: %v\n", err)
			return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
		}

		fmt.Printf("S3 upload successful. URL: %s\n", imageURL)
		product.Image = imageURL
	} else {
		fmt.Printf("No image file uploaded\n")
		// Tidak ada file yang diupload, lanjut tanpa image
	}

	// Create product waste data di database
	if err := configs.DB.Create(&product).Error; err != nil {
		// Jika upload foto berhasil tapi database gagal, hapus file dari S3
		if product.Image != "" {
			s3Service := helpers.NewS3Service()
			imageKey := s3Service.ExtractKeyFromURL(product.Image)
			if imageKey != "" {
				s3Service.DeleteFile(imageKey)
				fmt.Printf("Rollback: Deleted uploaded file from S3 due to database error\n")
			}
		}
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to create product waste: "+err.Error(), nil, nil)
	}

	// Jika ID product waste sudah tersedia, update nama file di S3 dengan ID yang benar (optional)
	if product.Image != "" && product.Id != 0 {
		fmt.Printf("Product waste created with ID: %d\n", product.Id)
	}

	return helpers.Response(c, 201, "Success", "Product waste created successfully", product, nil)
}

// UpdateProductWaste - Update product waste dengan upload foto ke AWS S3
func UpdateProductWaste(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid product ID", nil, nil)
	}

	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	var product models.ProductWaste
	if err := configs.DB.First(&product, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Product waste not found", nil, nil)
	}

	// Ambil values dari form
	wasteType := getFirstValue(form.Value["waste_type"])
	unit := getFirstValue(form.Value["unit"])
	category := getFirstValue(form.Value["category"])
	priceStr := getFirstValue(form.Value["price"])

	// Update fields jika provided
	if wasteType != "" {
		product.WasteType = wasteType
	}
	if unit != "" {
		product.Unit = unit
	}
	if category != "" {
		if category != "organik" && category != "anorganik" {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Category must be 'organik' or 'anorganik'", nil, nil)
		}
		product.Category = category
	}
	if priceStr != "" {
		price, err := strconv.Atoi(priceStr)
		if err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid price format", nil, nil)
		}
		product.Price = price
	}

	// Handle file upload jika ada
	files := form.File["image"]
	if len(files) > 0 {
		file := files[0]

		fmt.Printf("File received: %s, Size: %d bytes\n", file.Filename, file.Size)

		// Validate file size (max 2MB)
		if file.Size > 2<<20 {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "File size too large (max 2MB)", nil, nil)
		}

		// Validate file type
		allowedTypes := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedTypes[ext] {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
		}

		fmt.Printf("File type validated: %s\n", ext)

		// Initialize S3 service
		s3Service := helpers.NewS3Service()
		fmt.Printf("S3 service initialized\n")

		// Upload to S3
		fmt.Printf("Starting S3 upload for product waste image...\n")
		imageURL, err := s3Service.UploadFile(file, uint(id), "product-waste")
		if err != nil {
			fmt.Printf("S3 upload failed: %v\n", err)
			return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
		}

		// Delete old image from S3 jika ada
		if product.Image != "" {
			oldImageKey := s3Service.ExtractKeyFromURL(product.Image)
			if oldImageKey != "" {
				s3Service.DeleteFile(oldImageKey)
				fmt.Printf("Deleted old image from S3: %s\n", oldImageKey)
			}
		}

		fmt.Printf("S3 upload successful. URL: %s\n", imageURL)
		product.Image = imageURL
	} else {
		fmt.Printf("No new image file uploaded, keeping existing image\n")
	}

	// Update product waste data di database
	if err := configs.DB.Save(&product).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update product waste: "+err.Error(), nil, nil)
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
