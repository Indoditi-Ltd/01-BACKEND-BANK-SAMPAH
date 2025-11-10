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

// GetUserDashboard - Get all user data in one endpoint (profile + recent transactions) by email
func ScanBarcodeUser(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to read body", nil, nil)
	}

	// Get user data by email
	var user models.User
	if err := configs.DB.Preload("Plan").Where("email = ?", body.Email).First(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Get recent transactions (5 terakhir)
	var recentTransactions []models.HistoryModel
	if err := configs.DB.Where("user_id = ?", user.Id).
		Order("created_at DESC").
		Limit(5).
		Find(&recentTransactions).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch recent transactions", nil, nil)
	}

	// Format user profile
	profile := map[string]any{
		"id":                user.Id,
		"name":              user.Name,
		"phone":             user.Phone,
		"address":           user.Address,
		"photo":             user.Photo,
		"plan_name":         getPlanName(user.Plan),
		"balance":           user.Balance,
		"balance_formatted": formatBalance(user.Balance),
		"email":             user.Email,
	}

	// Format recent transactions menggunakan field yang sudah ada di HistoryModel
	var formattedTransactions []map[string]any
	for _, transaction := range recentTransactions {
		formattedTransactions = append(formattedTransactions, map[string]any{
			"id":               transaction.Id,
			"product_name":     transaction.ProductName,
			"product_type":     transaction.ProductType,
			"total_price":      transaction.TotalPrice,
			"amount_formatted": formatCurrency(transaction.TotalPrice),
			"date":             formatDate(transaction.CreatedAt),
			"status":           transaction.Status,
			"ref_id":           transaction.RefID,
			"user_number":      transaction.UserNumber,
		})
	}

	// Combine all data in one response
	dashboardData := map[string]any{
		"profile":             profile,
		"recent_transactions": formattedTransactions,
		"total_transactions":  len(recentTransactions),
	}

	return helpers.Response(c, 200, "Success", "User dashboard data retrieved successfully", dashboardData, nil)
}

// Helper functions
func getPlanName(plan *models.Plan) string {
	if plan != nil {
		return plan.Name
	}
	return "Mulungs Platinum" // Default value
}

func formatBalance(balance int) string {
	if balance >= 1000000 {
		return "Rp" + strconv.Itoa(balance/1000000) + ".000.000"
	}
	return "Rp" + strconv.Itoa(balance)
}
func formatCurrency(amount string) string {
	// Convert string amount to int for formatting
	if amount == "" {
		return "Rp0"
	}

	// Remove non-numeric characters and convert to int
	cleanAmount := strings.ReplaceAll(amount, ".", "")
	cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
	cleanAmount = strings.ReplaceAll(cleanAmount, "Rp", "")
	cleanAmount = strings.TrimSpace(cleanAmount)

	if num, err := strconv.Atoi(cleanAmount); err == nil {
		// Format dengan titik sebagai pemisah ribuan
		str := strconv.Itoa(num)
		var result string

		// Tambahkan titik setiap 3 digit dari belakang
		n := len(str)
		for i := 0; i < n; i++ {
			if i > 0 && (n-i)%3 == 0 {
				result += "."
			}
			result += string(str[i])
		}

		return "Rp" + result
	}

	return "Rp" + amount
}
func formatDate(date time.Time) string {
	// Format: "27 Agustus 2025"
	months := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return strconv.Itoa(date.Day()) + " " + months[date.Month()-1] + " " + strconv.Itoa(date.Year())
}
