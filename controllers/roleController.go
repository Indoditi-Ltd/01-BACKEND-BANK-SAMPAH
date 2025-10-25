package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"

	"github.com/gofiber/fiber/v2"
)

func ListRoleC(c *fiber.Ctx) error {
	var role []models.Role

	if err := configs.DB.Find(&role).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}
	return helpers.Response(c, 200, "Success", "Roles retrived successfully", role, nil)
}
