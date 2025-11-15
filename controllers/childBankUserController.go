package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateUserChildBank - Create new user child bank
func CreateUserChildBank(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		Province    string `json:"province"`
		District    string `json:"district"`
		ChildBankID uint   `json:"child_bank_id"` // Wajib untuk user child bank
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Validate required fields
	if body.Name == "" || body.Email == "" || body.Password == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Name, email, and password are required", nil, nil)
	}

	// Validasi ChildBankID harus ada
	if body.ChildBankID == 0 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Child bank ID is required", nil, nil)
	}

	// Cek apakah ChildBank exists
	var childBank models.ChildBank
	if err := configs.DB.First(&childBank, body.ChildBankID).Error; err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Child bank not found", nil, nil)
	}

	// Hash password
	hashedPassword, err := helpers.HashPassword(body.Password)
	if err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to hash password", nil, nil)
	}

	// Set role ID untuk user child bank (sesuaikan dengan ID role di database)
	roleID := uint(4) // Asumsi role ID 4 untuk user child bank

	user := models.User{
		Name:        body.Name,
		Email:       body.Email,
		Password:    hashedPassword,
		Phone:       body.Phone,
		Address:     body.Address,
		Province:    body.Province,
		District:    body.District,
		RoleID:      roleID,
		ChildBankID: &body.ChildBankID, // Gunakan pointer karena field nullable
		Status:      "active",
	}

	if err := configs.DB.Create(&user).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already registered", nil, nil)
		}
		return helpers.Response(c, 400, "Failed", err.Error(), nil, nil)
	}

	// Load relations untuk response
	configs.DB.Preload("Role").Preload("ChildBank").Preload("ChildBank.ParentBank").First(&user, user.Id)

	return helpers.Response(c, 200, "Success", "User child bank created successfully", user, nil)
}

// UpdateUserChildBank - Update user child bank
func UpdateUserChildBank(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	var body struct {
		Name        string `json:"name,omitempty"`
		Email       string `json:"email,omitempty"`
		Password    string `json:"password,omitempty"`
		Phone       string `json:"phone,omitempty"`
		Address     string `json:"address,omitempty"`
		Province    string `json:"province,omitempty"`
		District    string `json:"district,omitempty"`
		ChildBankID *uint  `json:"child_bank_id,omitempty"` // Pointer karena bisa null
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Cek apakah user exists dan role child bank
	var user models.User
	if err := configs.DB.Where("id = ? AND role_id = ?", id, 4).First(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User child bank not found", nil, nil)
	}

	// Validasi ChildBankID jika diupdate
	if body.ChildBankID != nil && *body.ChildBankID != 0 {
		var childBank models.ChildBank
		if err := configs.DB.First(&childBank, *body.ChildBankID).Error; err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Child bank not found", nil, nil)
		}
	}

	// Update fields
	updates := make(map[string]interface{})

	if body.Name != "" {
		updates["name"] = body.Name
	}

	if body.Email != "" {
		// Cek email unique untuk user lain
		var existingUser models.User
		if err := configs.DB.Where("email = ? AND id != ?", body.Email, id).First(&existingUser).Error; err == nil {
			return helpers.Response(c, 400, "Failed", "Email already registered by another user", nil, nil)
		}
		updates["email"] = body.Email
	}

	if body.Password != "" {
		hashedPassword, err := helpers.HashPassword(body.Password)
		if err != nil {
			return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to hash password", nil, nil)
		}
		updates["password"] = hashedPassword
	}

	if body.Phone != "" {
		updates["phone"] = body.Phone
	}

	if body.Address != "" {
		updates["address"] = body.Address
	}

	if body.Province != "" {
		updates["province"] = body.Province
	}

	if body.District != "" {
		updates["district"] = body.District
	}

	if body.ChildBankID != nil {
		updates["child_bank_id"] = body.ChildBankID
	}

	updates["updated_at"] = time.Now()

	if err := configs.DB.Model(&user).Updates(updates).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update user child bank", nil, nil)
	}

	// Load updated data dengan relations
	configs.DB.Preload("Role").Preload("ChildBank").Preload("ChildBank.ParentBank").First(&user, user.Id)

	return helpers.Response(c, 200, "Success", "User child bank updated successfully", user, nil)
}
func GetAllUsersChildBank(c *fiber.Ctx) error {
	parentBankID := c.Query("parent_bank_id")
	
	if parentBankID == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "parent_bank_id is required", nil, nil)
	}

	parentID, err := strconv.Atoi(parentBankID)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid parent_bank_id format", nil, nil)
	}

	var users []models.User
	
	// Query tanpa Preload dulu untuk test
	err = configs.DB.
		Joins("LEFT JOIN child_banks ON users.child_bank_id = child_banks.id").
		Where("users.role_id = ? AND child_banks.parent_bank_id = ?", 4, parentID).
		Find(&users).Error

	if err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch users: "+err.Error(), nil, nil)
	}

	if len(users) == 0 {
		return helpers.Response(c, 200, "Success", "No users found for this parent bank", []models.User{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Users child bank retrieved successfully", users, nil)
}

// DeleteUserChildBank - Delete user child bank (soft delete)
func DeleteUserChildBank(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	// Cek apakah user exists dan role child bank
	var user models.User
	if err := configs.DB.Where("id = ? AND role_id = ?", id, 4).First(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User child bank not found", nil, nil)
	}

	// Soft delete
	if err := configs.DB.Delete(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to delete user child bank", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "User child bank deleted successfully", nil, nil)
}
