package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GetUsers - Get all users with filtering and pagination (flat structure)
func GetUsersParentBank(c *fiber.Ctx) error {
	var query struct {
		ParentBankID *uint  `query:"parent_bank_id"`
		Search       string `query:"search"`
		Status       string `query:"status"`
		Page         int    `query:"page"`
		Limit        int    `query:"limit"`
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

	var users []models.User
	dbQuery := configs.DB.Preload("Division").Preload("Role").Preload("ParentBank")

	dbQuery = dbQuery.Where("role_id = ?", 3)

	if query.ParentBankID != nil {
		dbQuery = dbQuery.Where("parent_bank_id = ?", *query.ParentBankID)
	}

	if query.Search != "" {
		search := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("name LIKE ? OR email LIKE ?", search, search)
	}

	if query.Status != "" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	// Add ascending order by name
	dbQuery = dbQuery.Order("id DESC")

	var total int64
	dbQuery.Model(&models.User{}).Count(&total)

	if err := dbQuery.Offset(offset).Limit(query.Limit).Find(&users).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch users", nil, nil)
	}

	// Flat response structure
	response := map[string]interface{}{
		"users": users,
		"page":  query.Page,
		"limit": query.Limit,
		"total": total,
	}

	return helpers.Response(c, 200, "Success", "Users retrieved successfully", response, nil)
}

// CreateUserBankInduk - Create new user bank induk
func CreateUserBankInduk(c *fiber.Ctx) error {
	var body struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		Phone        string `json:"phone"`
		Address      string `json:"address"`
		Province     string `json:"province"` // Field baru
		District     string `json:"district"` // Field baru
		ParentBankID *uint  `json:"parent_bank_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Validate required fields berdasarkan UI
	if body.Name == "" || body.Email == "" || body.Password == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Name, email, and password are required", nil, nil)
	}

	// Hash password
	hashedPassword, err := helpers.HashPassword(body.Password)
	if err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to hash password", nil, nil)
	}

	// Set default role ID untuk user bank induk
	roleID := uint(3) // Sesuaikan dengan role ID untuk user bank induk

	user := models.User{
		Name:         body.Name,
		Email:        body.Email,
		Password:     hashedPassword,
		Phone:        body.Phone,
		Address:      body.Address,
		Province:     body.Province, // Field baru
		District:     body.District, // Field baru
		RoleID:       roleID,
		ParentBankID: body.ParentBankID,
		Status:       "active",
	}

	if err := configs.DB.Create(&user).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already registered", nil, nil)
		}
		return helpers.Response(c, 400, "Failed", err.Error(), nil, nil)
	}

	// Load relations untuk response
	configs.DB.Preload("Division").Preload("Role").Preload("ParentBank").First(&user, user.Id)

	return helpers.Response(c, 200, "Success", "User bank induk created successfully", user, nil)
}

// GetUserByID - Get user by ID
func GetUserByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	var user models.User
	if err := configs.DB.Preload("Division").Preload("Role").Preload("ParentBank").First(&user, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "User retrieved successfully", user, nil)
}

// UpdateUser - Update user
func UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	var body struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		Address      string `json:"address"`
		Province     string `json:"province"` // Field baru dari UI
		District     string `json:"district"` // Field baru dari UI
		DivisionID   *uint  `json:"division_id"`
		ParentBankID *uint  `json:"parent_bank_id"`
		Status       string `json:"status"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	var user models.User
	if err := configs.DB.First(&user, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Update fields
	user.Name = body.Name
	user.Email = body.Email
	user.Phone = body.Phone
	user.Address = body.Address
	user.Province = body.Province // Field baru
	user.District = body.District // Field baru
	user.DivisionID = body.DivisionID
	user.ParentBankID = body.ParentBankID
	if body.Status != "" {
		user.Status = body.Status
	}

	if err := configs.DB.Save(&user).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already exists", nil, nil)
		}
		return helpers.Response(c, 400, "Failed", err.Error(), nil, nil)
	}

	// Load relations for response
	configs.DB.Preload("Division").Preload("Role").Preload("ParentBank").First(&user, user.Id)

	return helpers.Response(c, 200, "Success", "User updated successfully", user, nil)
}

// DeleteUser - Soft delete user
func DeleteUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	var user models.User
	if err := configs.DB.First(&user, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	if err := configs.DB.Delete(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to delete user", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "User deleted successfully", nil, nil)
}

// ResetFilter - Reset filter (clear all filters)
func ResetFilter(c *fiber.Ctx) error {
	var users []models.User

	if err := configs.DB.Preload("Division").Preload("Role").Preload("ParentBank").
		Where("parent_bank_id IS NOT NULL").
		Find(&users).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch users", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Filter reset successfully", users, nil)
}

// GetParentBanks - Get all parent banks for dropdown
func GetParentBanksDropdown(c *fiber.Ctx) error {
	var parentBanks []models.ParentBank

	// Query untuk mendapatkan semua bank induk
	if err := configs.DB.Find(&parentBanks).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Could not find parent bank data", nil, nil)
	}

	// Jika tidak ada data, return array kosong
	if len(parentBanks) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.ParentBank{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Parent banks retrieved successfully", parentBanks, nil)
}
