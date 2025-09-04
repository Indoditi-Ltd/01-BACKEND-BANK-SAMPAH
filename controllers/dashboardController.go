package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"

	"github.com/gofiber/fiber/v2"
)

func DashboardTransactionCount(c *fiber.Ctx) error {
	var count int64
	err := configs.DB.Model(&models.Transaction{}).Count(&count).Error
	if err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get transaction count", nil, nil)
	}
	data := fiber.Map{
		"transaction_count": count,
	}
	return helpers.Response(c, fiber.StatusOK, "success", "Dashboard data fetched", data, nil)
}

