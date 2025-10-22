package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"

	"github.com/gofiber/fiber/v2"
)

func DashboardController(c *fiber.Ctx) error {
	var totalBankInduk int64
	var totalBankPembantu int64
	var totalMitra int64
	var totalUsers int64
	var company models.Company

	// Hitung total Bank Pembantu
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 2). // misal: 2 = Bank Pembantu
		Count(&totalUsers).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total bank pembantu", nil, nil)
	}
	// Hitung total Bank Pembantu
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 3). // misal: 2 = Bank Pembantu
		Count(&totalBankInduk).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total bank pembantu", nil, nil)
	}
	// Hitung total Bank Pembantu
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 4). // misal: 2 = Bank Pembantu
		Count(&totalBankPembantu).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total bank pembantu", nil, nil)
	}

	// Hitung total Mitra
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 5). // misal: 3 = Mitra
		Count(&totalMitra).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total mitra", nil, nil)
	}

	// Hitung total Users (admin atau semua user aktif)
	if err := configs.DB.Model(&company).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total users", nil, nil)
	}
	data := fiber.Map{
		"total_user":            totalUsers,
		"total_parent_bank":     totalBankInduk,
		"total_child_bank":      totalBankPembantu,
		"total_mitra":           totalMitra,
		"total_uang_perusahaan": company.Balance,
	}

	return helpers.Response(c, fiber.StatusOK, "success", "Dashboard data fetched", data, nil)
}
