package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RegisterUser(c *fiber.Ctx) error {
	var body struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body"+err.Error(), nil, nil)
	}

	hashedPassword, err := helpers.HashPassword(body.Password)

	if err != nil {
		return err
	}

	planID := uint(2)

	endUser := models.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hashedPassword,
		PlanID:   &planID,
		RoleID:   2,
	}

	if err := configs.DB.Create(&endUser).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, 400, "Failed", "Email already registered", nil, nil)
		}
		return helpers.Response(c, 400, "Failed", err.Error(), nil, nil)
	}
	return helpers.Response(c, 200, "Success", "Admin create successfully", endUser, nil)
}
