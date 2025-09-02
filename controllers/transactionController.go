package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"

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
	var transaction []models.Transaction
	if err := configs.DB.Preload("User").Preload("User.Plan").Preload("Admin").Find(&transaction, "type = ?", "topup").Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to get data top up", nil, nil)
	}

	if len(transaction) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.Transaction{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data found", transaction, nil)
}

func TransactionAllWithdraw(c *fiber.Ctx) error {
	var transaction []models.Transaction
	if err := configs.DB.Preload("User").Preload("User.Plan").Preload("Admin").Find(&transaction, "type = ?", "withdraw").Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to get data top up", nil, nil)
	}

	if len(transaction) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.Transaction{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data found", transaction, nil)
}
