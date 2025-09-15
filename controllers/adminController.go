package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetAllAdmin(c *fiber.Ctx) error {
	var admins []models.User
	if err := configs.DB.Preload("Role").Preload("Division").Find(&admins, "role_id = ?", "1").Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Could not find admin", nil, nil)
	}

	if len(admins) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.User{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data Found", admins, nil)
}

func GetAdminByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var userAdmin models.User

	if err := configs.DB.Preload("Role").Preload("Division").Find(&userAdmin, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Admin not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}
	return helpers.Response(c, 200, "Success", "Admin retrieved successfully", userAdmin, nil)
}

func CreateAdmin(c *fiber.Ctx) error {
	var body struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		Address    string `json:"address"`
		Photo      string `json:"photo"`
		DivisionID *uint  `json:"division_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	hashedPassword, err := helpers.HashPassword("Admin123")
	if err != nil {
		return err
	}

	userAdmin := models.User{
		Name:       body.Name,
		Email:      body.Email,
		Phone:      body.Phone,
		Address:    body.Address,
		Photo:      body.Photo,
		DivisionID: body.DivisionID,
		RoleID:     1,
		Password:   hashedPassword,
	}

	if err := configs.DB.Create(&userAdmin).Error; err != nil {
		// return helpers.Response(c, 400, "Failed", "Failed to create admin", nil, nil)
		// Cek apakah error duplicate
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already registered", nil, nil)
		}
		return helpers.Response(c, 400, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Admin create successfully", userAdmin, nil)
}

// Function for Admin in Web (Update Admin)
func UpdateUserAdmin(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		Address    string `json:"address"`
		Photo      string `json:"photo"`
		Password   string `json:"password"`
		DivisionID *uint  `json:"division_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	var userAdmin models.User
	if err := configs.DB.First(&userAdmin, id).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Admin not found", nil, nil)
	}

	userAdmin.Name = body.Name
	userAdmin.Email = body.Email
	userAdmin.Address = body.Address
	userAdmin.Phone = body.Phone
	userAdmin.Photo = body.Photo
	userAdmin.DivisionID = body.DivisionID

	if body.Password != "" {
		hashedPassword, err := helpers.HashPassword(body.Password)
		if err != nil {
			return helpers.Response(c, 500, "Failed", "Error hashing password", nil, nil)
		}
		userAdmin.Password = hashedPassword
	}

	if err := configs.DB.Save(&userAdmin).Error; err != nil {
		// return helpers.Response(c, 500, "Failed", "Failed to update Admin", nil, nil)
		// Cek apakah error duplicate
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already registered", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Admin updated successfully", userAdmin, nil)
}

func DeleteUserAdmin(c *fiber.Ctx) error {
	id := c.Params("id")

	var userAdmin models.User

	if err := configs.DB.First(&userAdmin, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Admin not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	if err := configs.DB.Delete(&userAdmin).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Admin deleted successfully", nil, nil)
}
