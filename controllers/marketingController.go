package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateMarketing - Create new marketing data dengan upload foto ke AWS S3
func CreateMarketing(c *fiber.Ctx) error {
	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form
	title := getFirstValue(form.Value["title"])
	startDateStr := getFirstValue(form.Value["start_date"])
	endDateStr := getFirstValue(form.Value["end_date"])
	broadcast := getFirstValue(form.Value["broadcast"])
	description := getFirstValue(form.Value["description"])

	// Validate required fields
	if title == "" || startDateStr == "" || endDateStr == "" || description == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Title, start date, end date, and description are required", nil, nil)
	}

	// Set location to UTC untuk menghindari timezone shift
	loc, _ := time.LoadLocation("UTC")

	// Parse dates dengan format: "dd/mm/yyyy HH:mm" (24-hour format)
	startDate, err := time.ParseInLocation("02/01/2006 15:04", startDateStr, loc)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid start date format (dd/mm/yyyy HH:mm)", nil, nil)
	}

	endDate, err := time.ParseInLocation("02/01/2006 15:04", endDateStr, loc)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid end date format (dd/mm/yyyy HH:mm)", nil, nil)
	}

	// Validate dates
	if endDate.Before(startDate) {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "End date cannot be before start date", nil, nil)
	}

	// Initialize marketing dengan data dasar
	marketing := models.Marketing{
		Title:       title,
		StartDate:   startDate,
		EndDate:     endDate,
		Broadcast:   broadcast,
		Description: description,
		Image:       "", // Default empty, akan diupdate jika ada upload foto
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

		// Initialize S3 service - gunakan cara yang sama seperti di UploadProfilePhoto
		s3Service := helpers.NewS3Service()
		fmt.Printf("S3 service initialized\n")

		// Upload to S3 - menggunakan ID marketing 0 (sementara)
		fmt.Printf("Starting S3 upload for marketing image...\n")
		imageURL, err := s3Service.UploadFile(file, 0, "marketing")
		if err != nil {
			fmt.Printf("S3 upload failed: %v\n", err)
			return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
		}

		fmt.Printf("S3 upload successful. URL: %s\n", imageURL)
		marketing.Image = imageURL
	} else {
		fmt.Printf("No image file uploaded\n")
		// Tidak ada file yang diupload, lanjut tanpa image
	}

	// Create marketing data di database
	if err := configs.DB.Create(&marketing).Error; err != nil {
		// Jika upload foto berhasil tapi database gagal, hapus file dari S3
		if marketing.Image != "" {
			s3Service := helpers.NewS3Service()
			imageKey := s3Service.ExtractKeyFromURL(marketing.Image)
			if imageKey != "" {
				s3Service.DeleteFile(imageKey)
				fmt.Printf("Rollback: Deleted uploaded file from S3 due to database error\n")
			}
		}
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to create marketing data: "+err.Error(), nil, nil)
	}

	// Jika ID marketing sudah tersedia, update nama file di S3 dengan ID yang benar (optional)
	if marketing.Image != "" && marketing.Id != 0 {
		fmt.Printf("Marketing created with ID: %d\n", marketing.Id)
	}

	return helpers.Response(c, 201, "Success", "Marketing data created successfully", marketing, nil)
}

// Helper function untuk mengambil first value dari string slice
func getFirstValue(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
// GetMarketingList - Get all marketing data with filtering (tanpa category dan status)
func GetMarketingList(c *fiber.Ctx) error {
	var query struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Search    string `query:"search"`
		Page      int    `query:"page"`
		Limit     int    `query:"limit"`
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

	var marketing []models.Marketing
	dbQuery := configs.DB

	// Apply date filter
	if query.StartDate != "" && query.EndDate != "" {
		startDate, err1 := time.Parse("02/01/2006", query.StartDate)
		endDate, err2 := time.Parse("02/01/2006", query.EndDate)

		if err1 == nil && err2 == nil {
			// Adjust end date to end of day
			endDate = endDate.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
			dbQuery = dbQuery.Where("start_date >= ? AND end_date <= ?", startDate, endDate)
		}
	}

	// Apply search filter (by title/description/broadcast)
	if query.Search != "" {
		search := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR description LIKE ? OR broadcast LIKE ?", search, search, search)
	}

	var total int64
	dbQuery.Model(&models.Marketing{}).Count(&total)

	// Calculate total pages
	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	// Get data dengan pagination dan order terbaru
	if err := dbQuery.Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch marketing data", nil, nil)
	}

	// Format response
	var formattedMarketing []map[string]any
	for i, item := range marketing {
		formattedMarketing = append(formattedMarketing, map[string]any{
			"no":          i + 1 + offset,
			"id":          item.Id,
			"image":       item.Image,
			"title":       item.Title,
			"start_date":  helpers.FormatDateWithTime(item.StartDate),
			"end_date":    helpers.FormatDateWithTime(item.EndDate),
			"broadcast":   item.Broadcast,
			"description": item.Description,
			"created_at":  item.CreatedAt,
			"updated_at":  item.UpdatedAt,
		})
	}

	// Format meta data
	meta := map[string]any{
		"limit": query.Limit,
		"page":  query.Page,
		"pages": totalPages,
		"total": total,
	}

	response := map[string]any{
		"marketing": formattedMarketing,
		"meta":      meta,
	}

	return helpers.Response(c, 200, "Success", "Marketing data retrieved successfully", response, nil)
}

// UpdateMarketing - Update marketing data dengan upload foto ke AWS S3
func UpdateMarketing(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid marketing ID", nil, nil)
	}

	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form
	title := getFirstValue(form.Value["title"])
	startDateStr := getFirstValue(form.Value["start_date"])
	endDateStr := getFirstValue(form.Value["end_date"])
	broadcast := getFirstValue(form.Value["broadcast"])
	description := getFirstValue(form.Value["description"])

	// Cari marketing data yang akan diupdate
	var marketing models.Marketing
	if err := configs.DB.First(&marketing, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Marketing data not found", nil, nil)
	}

	// Simpan old image URL untuk delete nanti jika ada upload baru
	oldImageURL := marketing.Image

	// Set location to UTC
	loc, _ := time.LoadLocation("UTC")

	// Update fields
	if title != "" {
		marketing.Title = title
	}

	if startDateStr != "" {
		startDate, err := time.ParseInLocation("02/01/2006 15:04", startDateStr, loc)
		if err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid start date format (dd/mm/yyyy HH:mm)", nil, nil)
		}
		marketing.StartDate = startDate
	}

	if endDateStr != "" {
		endDate, err := time.ParseInLocation("02/01/2006 15:04", endDateStr, loc)
		if err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid end date format (dd/mm/yyyy HH:mm)", nil, nil)
		}
		marketing.EndDate = endDate
	}

	// Validate dates jika kedua date diupdate
	if startDateStr != "" && endDateStr != "" {
		if marketing.EndDate.Before(marketing.StartDate) {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "End date cannot be before start date", nil, nil)
		}
	}

	if broadcast != "" {
		marketing.Broadcast = broadcast
	}
	if description != "" {
		marketing.Description = description
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
		fmt.Printf("Starting S3 upload for marketing image...\n")
		imageURL, err := s3Service.UploadFile(file, uint(id), "marketing")
		if err != nil {
			fmt.Printf("S3 upload failed: %v\n", err)
			return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
		}

		fmt.Printf("S3 upload successful. URL: %s\n", imageURL)
		marketing.Image = imageURL

		// Delete old image from S3 jika ada
		if oldImageURL != "" {
			fmt.Printf("Deleting old image: %s\n", oldImageURL)
			oldKey := s3Service.ExtractKeyFromURL(oldImageURL)
			if oldKey != "" {
				if err := s3Service.DeleteFile(oldKey); err != nil {
					fmt.Printf("Warning: Failed to delete old file: %v\n", err)
					// Continue update even if delete old file fails
				} else {
					fmt.Printf("Old image deleted successfully\n")
				}
			}
		}
	}

	// Update marketing data di database
	if err := configs.DB.Save(&marketing).Error; err != nil {
		// Jika upload foto baru berhasil tapi database gagal, hapus file baru dari S3
		if marketing.Image != oldImageURL && marketing.Image != "" {
			s3Service := helpers.NewS3Service()
			newImageKey := s3Service.ExtractKeyFromURL(marketing.Image)
			if newImageKey != "" {
				s3Service.DeleteFile(newImageKey)
				fmt.Printf("Rollback: Deleted uploaded file from S3 due to database error\n")
			}
		}
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update marketing data: "+err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Marketing data updated successfully", marketing, nil)
}

// DeleteMarketing - Delete marketing data
func DeleteMarketing(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid marketing ID", nil, nil)
	}

	var marketing models.Marketing
	if err := configs.DB.First(&marketing, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Marketing data not found", nil, nil)
	}

	if err := configs.DB.Delete(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to delete marketing data", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Marketing data deleted successfully", nil, nil)
}
