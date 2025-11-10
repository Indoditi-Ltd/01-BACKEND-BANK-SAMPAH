package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetDonationList - Get all donation data with pagination and filters
func GetDonationListAll(c *fiber.Ctx) error {
	var req struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Search    string `query:"search"`
		Status    string `query:"status"`
		Page      int    `query:"page"`
		Limit     int    `query:"limit"`
	}

	// Parse query parameters
	if err := c.QueryParser(&req); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to parse query parameters", nil, nil)
	}

	// Set default values
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit

	// Build query
	query := configs.DB.Model(&models.Donation{}).Where("deleted_at IS NULL")

	// Apply date filter
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			query = query.Where("DATE(created_at) >= ?", startDate.Format("2006-01-02"))
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			query = query.Where("DATE(created_at) <= ?", endDate.Format("2006-01-02"))
		}
	}

	// Apply status filter
	if req.Status != "" {
		query = query.Where("status = ?", "pending")
	}

	// Apply search filter (description)
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("description ILIKE ?", searchPattern)
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get donations dengan pagination
	var donations []models.Donation
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&donations).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]any{
		"donations": donations,
		"meta": map[string]any{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Donation data retrieved successfully", data, nil)
}

// GetDonationList - Get all donation data with pagination and filters
func GetDonationList(c *fiber.Ctx) error {
	var req struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Search    string `query:"search"`
		Status    string `query:"status"`
		Page      int    `query:"page"`
		Limit     int    `query:"limit"`
	}

	// Parse query parameters
	if err := c.QueryParser(&req); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to parse query parameters", nil, nil)
	}

	// Set default values
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit

	// Build query - HANYA TAMPILKAN YANG STATUS = 'pending'
	query := configs.DB.Model(&models.Donation{}).Where("deleted_at IS NULL AND status = ?", "pending")

	// Apply date filter
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			query = query.Where("DATE(created_at) >= ?", startDate.Format("2006-01-02"))
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			query = query.Where("DATE(created_at) <= ?", endDate.Format("2006-01-02"))
		}
	}

	// Apply status filter
	if req.Status != "" {
		query = query.Where("status = ?", "pending")
	}

	// Apply search filter (description)
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("description ILIKE ?", searchPattern)
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get donations dengan pagination
	var donations []models.Donation
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&donations).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]any{
		"donations": donations,
		"meta": map[string]any{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Donation data retrieved successfully", data, nil)
}

// GetDonationByID - Get single donation data by ID
func GetDonationByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var donation models.Donation
	result := configs.DB.Where("id = ? AND deleted_at IS NULL", id).First(&donation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Donation data not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Donation data retrieved successfully", donation, nil)
}

// CreateDonation - Create new donation data dengan upload foto ke AWS S3
func CreateDonation(c *fiber.Ctx) error {
	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form
	startDateStr := getFirstValue(form.Value["start_date"])
	endDateStr := getFirstValue(form.Value["end_date"])
	description := getFirstValue(form.Value["description"])
	currentAmountStr := getFirstValue(form.Value["current_amount"])

	// Validate required fields
	if startDateStr == "" || endDateStr == "" || description == "" {
		return helpers.Response(c, 400, "Failed", "Start date, end date, and description are required", nil, nil)
	}

	// Parse dates dengan format: "dd/mm/yyyy" (sesuai form)
	startDate, err := time.Parse("02/01/2006", startDateStr)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid start date format (dd/mm/yyyy)", nil, nil)
	}

	endDate, err := time.Parse("02/01/2006", endDateStr)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid end date format (dd/mm/yyyy)", nil, nil)
	}

	// Validate dates
	if endDate.Before(startDate) {
		return helpers.Response(c, 400, "Failed", "End date cannot be before start date", nil, nil)
	}

	// Parse current amount jika ada
	currentAmount := 0
	if currentAmountStr != "" {
		if amount, err := strconv.Atoi(currentAmountStr); err == nil {
			currentAmount = amount
		}
	}

	// Initialize donation dengan data dasar
	donation := models.Donation{
		StartDate:     startDate,
		EndDate:       endDate,
		Description:   description,
		CurrentAmount: currentAmount,
		Status:        "pending", // Default status
		Image:         "",        // Default empty, akan diupdate jika ada upload foto
	}

	// Handle file upload jika ada
	files := form.File["image"]
	if len(files) > 0 {
		file := files[0]

		// Validate file size (max 2MB)
		if file.Size > 2<<20 {
			return helpers.Response(c, 400, "Failed", "File size too large (max 2MB)", nil, nil)
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
			return helpers.Response(c, 400, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
		}

		// Initialize S3 service
		s3Service := helpers.NewS3Service()

		// Upload to S3 - menggunakan ID donation 0 (sementara)
		imageURL, err := s3Service.UploadFile(file, 0, "donation")
		if err != nil {
			return helpers.Response(c, 500, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
		}

		donation.Image = imageURL
	}

	// Create donation data di database
	if err := configs.DB.Create(&donation).Error; err != nil {
		// Jika upload foto berhasil tapi database gagal, hapus file dari S3
		if donation.Image != "" {
			s3Service := helpers.NewS3Service()
			imageKey := s3Service.ExtractKeyFromURL(donation.Image)
			if imageKey != "" {
				s3Service.DeleteFile(imageKey)
			}
		}
		return helpers.Response(c, 500, "Failed", "Failed to create donation data: "+err.Error(), nil, nil)
	}

	return helpers.Response(c, 201, "Success", "Donation data created successfully", donation, nil)
}

// UpdateDonation - Update donation data dengan upload foto baru ke AWS S3
func UpdateDonation(c *fiber.Ctx) error {
	id := c.Params("id")

	// Cek apakah donation data exists
	var existingDonation models.Donation
	result := configs.DB.Where("id = ? AND deleted_at IS NULL", id).First(&existingDonation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Donation data not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form (gunakan existing value jika tidak diupdate)
	startDateStr := getFirstValue(form.Value["start_date"])
	if startDateStr != "" {
		startDate, err := time.Parse("02/01/2006", startDateStr)
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Invalid start date format (dd/mm/yyyy)", nil, nil)
		}
		existingDonation.StartDate = startDate
	}

	endDateStr := getFirstValue(form.Value["end_date"])
	if endDateStr != "" {
		endDate, err := time.Parse("02/01/2006", endDateStr)
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Invalid end date format (dd/mm/yyyy)", nil, nil)
		}
		existingDonation.EndDate = endDate
	}

	description := getFirstValue(form.Value["description"])
	if description != "" {
		existingDonation.Description = description
	}

	currentAmountStr := getFirstValue(form.Value["current_amount"])
	if currentAmountStr != "" {
		if amount, err := strconv.Atoi(currentAmountStr); err == nil {
			existingDonation.CurrentAmount = amount
		}
	}

	status := getFirstValue(form.Value["status"])
	if status != "" && (status == "pending" || status == "completed" || status == "cancelled") {
		existingDonation.Status = status
	}

	// Validate dates jika keduanya diupdate
	if !existingDonation.EndDate.After(existingDonation.StartDate) {
		return helpers.Response(c, 400, "Failed", "End date cannot be before start date", nil, nil)
	}

	// Handle file upload jika ada gambar baru
	files := form.File["image"]
	if len(files) > 0 {
		file := files[0]

		// Validate file size (max 2MB)
		if file.Size > 2<<20 {
			return helpers.Response(c, 400, "Failed", "File size too large (max 2MB)", nil, nil)
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
			return helpers.Response(c, 400, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
		}

		// Initialize S3 service
		s3Service := helpers.NewS3Service()

		// Hapus gambar lama jika ada
		if existingDonation.Image != "" {
			oldImageKey := s3Service.ExtractKeyFromURL(existingDonation.Image)
			if oldImageKey != "" {
				s3Service.DeleteFile(oldImageKey)
			}
		}

		// Upload gambar baru ke S3
		imageURL, err := s3Service.UploadFile(file, existingDonation.Id, "donation")
		if err != nil {
			return helpers.Response(c, 500, "Failed", "Failed to upload new image to cloud storage: "+err.Error(), nil, nil)
		}

		existingDonation.Image = imageURL
	}

	// Update donation data di database
	if err := configs.DB.Save(&existingDonation).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to update donation data: "+err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Donation data updated successfully", existingDonation, nil)
}

// DeleteDonation - Soft delete donation data
func DeleteDonation(c *fiber.Ctx) error {
	id := c.Params("id")

	// Cek apakah donation data exists
	var donation models.Donation
	result := configs.DB.Where("id = ? AND deleted_at IS NULL", id).First(&donation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Donation data not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// Hapus gambar dari S3 jika ada
	if donation.Image != "" {
		s3Service := helpers.NewS3Service()
		imageKey := s3Service.ExtractKeyFromURL(donation.Image)
		if imageKey != "" {
			s3Service.DeleteFile(imageKey)
		}
	}

	// Soft delete dari database
	if err := configs.DB.Delete(&donation).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to delete donation data: "+err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Donation data deleted successfully", nil, nil)
}
