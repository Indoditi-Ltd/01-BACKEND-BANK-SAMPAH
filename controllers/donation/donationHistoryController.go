package donation

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CreateDonation - Create donation (langsung potong saldo & catat history)
func CreateDonation(c *fiber.Ctx) error {
	var req struct {
		DonationID uint `json:"donation_id" validate:"required"`
		Amount     int  `json:"amount" validate:"required,min=1000"` // minimal donation 1000
	}

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid request body", nil, nil)
	}

	// Get user_id from parameter
	userID := c.Params("user_id")
	if userID == "" {
		return helpers.Response(c, 400, "Failed", "User ID is required", nil, nil)
	}

	// Convert userID to uint
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid user ID format", nil, nil)
	}

	// Start database transaction
	tx := configs.DB.Begin()

	// 1. Get user data dengan lock untuk avoid race condition
	var user models.User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, uint(userIDUint)).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
	}

	// 2. Validasi donation exists dan masih aktif
	var donation models.Donation
	if err := tx.Where("id = ? AND deleted_at IS NULL", req.DonationID).First(&donation).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Donation campaign not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// 3. Validasi status dan tanggal donation
	now := time.Now()
	if donation.Status != "pending" {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Donation campaign is not active", nil, nil)
	}
	if now.Before(donation.StartDate) {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Donation campaign has not started yet", nil, nil)
	}
	if now.After(donation.EndDate) {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Donation campaign has ended", nil, nil)
	}

	// 4. Validasi saldo user cukup
	if user.Balance < req.Amount {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Insufficient balance", nil, nil)
	}

	// 5. Potong saldo user
	user.Balance -= req.Amount
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to deduct user balance", nil, nil)
	}

	// 6. Auto increment currentAmount di donation
	donation.CurrentAmount += req.Amount
	if err := tx.Save(&donation).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to update donation amount", nil, nil)
	}

	// 7. Create donation history
	history := models.DonationHistory{
		UserID:     uint(userIDUint),
		DonationID: req.DonationID,
		Amount:     req.Amount,
	}

	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to create donation history", nil, nil)
	}

	// 8. Commit transaction
	tx.Commit()

	// Preload data untuk response
	configs.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, email")
	}).Preload("Donation").First(&history, history.Id)

	return helpers.Response(c, 201, "Success", "Donation successful", history, nil)
}

// GetUserDonations - Get donation history for logged in user
func GetUserDonations(c *fiber.Ctx) error {
	// Get user_id from parameter
	userID := c.Params("user_id")
	if userID == "" {
		return helpers.Response(c, 400, "Failed", "User ID is required", nil, nil)
	}

	// Convert userID to uint
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid user ID format", nil, nil)
	}

	var req struct {
		Page  int `query:"page"`
		Limit int `query:"limit"`
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
	query := configs.DB.Model(&models.DonationHistory{}).
		Preload("Donation").
		Where("user_id = ?", uint(userIDUint))

	// Get total count
	var total int64
	query.Count(&total)

	// Get donations history dengan pagination
	var donations []models.DonationHistory
	err = query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&donations).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation history", nil, nil)
	}

	// Format response
	data := map[string]any{
		"donations": donations,
		"meta": map[string]any{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Donation history retrieved successfully", data, nil)
}

// CompleteDonation - Mengosongkan amount donasi dan ubah status menjadi completed
func CompleteDonation(c *fiber.Ctx) error {
	// Get donation_id from parameter
	donationID := c.Params("donation_id")
	if donationID == "" {
		return helpers.Response(c, 400, "Failed", "Donation ID is required", nil, nil)
	}

	// Convert donationID to uint
	donationIDUint, err := strconv.ParseUint(donationID, 10, 32)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid donation ID format", nil, nil)
	}

	// Get user_id from parameter
	userID := c.Params("user_id")
	if userID == "" {
		return helpers.Response(c, 400, "Failed", "User ID is required", nil, nil)
	}

	// Convert userID to uint
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid user ID format", nil, nil)
	}

	// Start database transaction
	tx := configs.DB.Begin()

	// 1. Validasi donation exists
	var donation models.Donation
	if err := tx.Where("id = ? AND deleted_at IS NULL", uint(donationIDUint)).First(&donation).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Donation campaign not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Failed to fetch donation data", nil, nil)
	}

	// 2. Validasi donation belum completed
	if donation.Status == "completed" {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Donation campaign already completed", nil, nil)
	}

	// 3. Simpan amount sebelumnya untuk history
	previousAmount := donation.CurrentAmount

	// 4. Update donation - set amount 0 dan status completed
	donation.CurrentAmount = 0
	donation.Status = "completed"

	if err := tx.Save(&donation).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to complete donation", nil, nil)
	}

	// 5. Create completion history (opsional, untuk audit trail)
	completionHistory := models.DonationHistory{
		UserID:     uint(userIDUint), // user yang melakukan complete
		DonationID: uint(donationIDUint),
		Amount:     0, // amount 0 untuk menandai completion
	}

	if err := tx.Create(&completionHistory).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to create completion history", nil, nil)
	}

	// 6. Commit transaction
	tx.Commit()

	response := map[string]interface{}{
		"donation": map[string]interface{}{
			"id":             donation.Id,
			"current_amount": donation.CurrentAmount,
			"status":         donation.Status,
			"previous_amount": previousAmount,
		},
		"completed_by": userIDUint,
	}

	return helpers.Response(c, 200, "Success", "Donation completed successfully", response, nil)
}