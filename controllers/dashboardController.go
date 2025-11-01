package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"strings"

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
	// Hitung total Bank Induk
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 3). // misal: 3 = Bank Induk
		Count(&totalBankInduk).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total bank induk", nil, nil)
	}
	// Hitung total Bank Pembantu
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 4). // misal: 4 = Bank Pembantu
		Count(&totalBankPembantu).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total bank pembantu", nil, nil)
	}
	// Hitung total Mitra
	if err := configs.DB.Model(&models.User{}).
		Where("role_id = ?", 5). // misal: 5 = Mitra
		Count(&totalMitra).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get total mitra", nil, nil)
	}

	// Get company balance
	if err := configs.DB.First(&company).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get company balance", nil, nil)
	}

	// Ambil data transaksi terbaru (latest activity)
	var latestTransactions []models.Transaction

	// Query untuk mendapatkan data transaksi terbaru dengan preload user
	if err := configs.DB.
		Preload("User").
		Where("status = ?", "pending"). // Hanya transaksi yang sudah dikonfirmasi
		Order("created_at DESC").
		Limit(7). // Ambil 7 data terbaru sesuai gambar
		Find(&latestTransactions).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "error", "Failed to get latest transactions", nil, nil)
	}

	// Format response untuk latest activity
	var latestActivity []map[string]interface{}

	for i, transaction := range latestTransactions {
		// Format status dengan ikon
		var statusDisplay string
		switch transaction.Type {
		case "withdraw":
			statusDisplay = "Penarikan"
		case "topup":
			statusDisplay = "Top up"
		default:
			statusDisplay = transaction.Type
		}

		// Format tanggal
		formattedDate := transaction.CreatedAt.Format("02-01-2006")

		activity := map[string]interface{}{
			"no":        i + 1,
			"name_user": transaction.User.Name,
			"total":     formatCurrency(transaction.Balance),
			"tanggal":   formattedDate,
			"email":     transaction.User.Email,
			"status":    statusDisplay,
		}

		latestActivity = append(latestActivity, activity)
	}

	data := fiber.Map{
		"total_user":            totalUsers,
		"total_parent_bank":     totalBankInduk,
		"total_child_bank":      totalBankPembantu,
		"total_mitra":           totalMitra,
		"total_uang_perusahaan": company.Balance,
		"latest_activity":       latestActivity,
	}

	return helpers.Response(c, fiber.StatusOK, "success", "Dashboard data fetched", data, nil)
}

// Helper function untuk format currency
func formatCurrency(amount int) string {
	// Konversi ke float untuk formatting
	floatAmount := float64(amount)

	// Format dengan separator ribuan dan 2 digit desimal
	str := fmt.Sprintf("Rp %.2f", floatAmount)

	// Tambahkan separator ribuan
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = "." + parts[1]
	}

	// Format ribuan (skip "Rp " di depan)
	rpPrefix := "Rp "
	numberPart := integerPart[len(rpPrefix):]

	var formattedInteger string
	count := 0
	// Balik string untuk memudahkan penambahan titik
	for i := len(numberPart) - 1; i >= 0; i-- {
		if count > 0 && count%3 == 0 {
			formattedInteger = "." + formattedInteger
		}
		formattedInteger = string(numberPart[i]) + formattedInteger
		count++
	}

	return rpPrefix + formattedInteger + decimalPart
}
