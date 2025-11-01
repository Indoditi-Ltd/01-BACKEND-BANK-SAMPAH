package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TransactionCreateTopUp(c *fiber.Ctx) error {
	var body struct {
		UserID  uint   `json:"user_id"`
		Balance int    `json:"balance"`
		AdminID uint   `json:"admin_id"`
		Desc    string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	transaction := models.Transaction{
		UserID:  body.UserID,
		Balance: body.Balance,
		Status:  "pending",
		Desc:    body.Desc,
		Type:    "topup",
	}

	if err := configs.DB.Create(&transaction).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to create transaction", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Transaction create successfully", transaction, nil)
}

func TransactionCreateWithdraw(c *fiber.Ctx) error {
	var body struct {
		UserID  uint   `json:"user_id"`
		Balance int    `json:"balance"`
		Desc    string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	transaction := models.Transaction{
		UserID:  body.UserID,
		Balance: body.Balance,
		Status:  "pending",
		Desc:    body.Desc,
		Type:    "withdraw",
	}

	if err := configs.DB.Create(&transaction).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to create transaction", nil, nil)
	}

	res := models.TransactionCreateResponse{
		UserID:  transaction.UserID,
		Balance: transaction.Balance,
		Type:    transaction.Type,
		Status:  transaction.Status,
		Desc:    transaction.Desc,
	}

	return helpers.Response(c, 200, "Success", "Transaction create successfully", res, nil)
}

func TransactionAllTopUp(c *fiber.Ctx) error {
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
	query := configs.DB.Model(&models.Transaction{}).
		Preload("User").
		Preload("User.Plan").
		Preload("Admin").
		Where("type = ?", "topup")

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
		query = query.Where("status = ?", req.Status)
	}

	// Apply search filter (nama/email user) - CARA AMAN
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		
		// Cari user yang sesuai dengan search pattern
		var userIDs []uint
		configs.DB.Model(&models.User{}).
			Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
			Pluck("id", &userIDs)
		
		// Jika ada user yang ditemukan, filter transaksi berdasarkan user_id
		if len(userIDs) > 0 {
			query = query.Where("user_id IN (?)", userIDs)
		} else {
			// Jika tidak ada user yang cocok, return empty result
			data := map[string]interface{}{
				"transactions": []models.Transaction{},
				"meta": map[string]interface{}{
					"page":  req.Page,
					"limit": req.Limit,
					"total": 0,
					"pages": 0,
				},
			}
			return helpers.Response(c, 200, "Success", "Data found", data, nil)
		}
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get transactions dengan pagination
	var transactions []models.Transaction
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&transactions).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch topup transactions", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]interface{}{
		"transactions": transactions,
		"meta": map[string]interface{}{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Data found", data, nil)
}

func TransactionAllWithdraw(c *fiber.Ctx) error {
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
	query := configs.DB.Model(&models.Transaction{}).
		Preload("User").
		Preload("User.Plan").
		Preload("Admin").
		Where("type = ?", "withdraw")

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
		query = query.Where("status = ?", req.Status)
	}

	// Apply search filter (nama/email user) - PERBAIKAN DI SINI
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"

		// Cari user yang sesuai dengan search pattern
		var userIDs []uint
		configs.DB.Model(&models.User{}).
			Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
			Pluck("id", &userIDs)

		// Jika ada user yang ditemukan, filter transaksi berdasarkan user_id
		if len(userIDs) > 0 {
			query = query.Where("user_id IN (?)", userIDs)
		} else {
			// Jika tidak ada user yang cocok, return empty result
			data := map[string]interface{}{
				"transactions": []models.Transaction{},
				"meta": map[string]interface{}{
					"page":  req.Page,
					"limit": req.Limit,
					"total": 0,
					"pages": 0,
				},
			}
			return helpers.Response(c, 200, "Success", "Data found", data, nil)
		}
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get transactions dengan pagination
	var transactions []models.Transaction
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&transactions).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch withdrawal transactions", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]interface{}{
		"transactions": transactions,
		"meta": map[string]interface{}{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Data found", data, nil)
}
