package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Function for admin in web
func GetPrentBank(c *fiber.Ctx) error {
	var parentBank []models.ParentBank
	if err := configs.DB.Find(&parentBank).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Could not find parent bank data", nil, nil)
	}

	if len(parentBank) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.ParentBank{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data Found", parentBank, nil)
}

// Function for admin in web
func CreateParentBank(c *fiber.Ctx) error {
	var body struct {
		District string `json:"district"`
		Province string `json:"province"`
		Address  string `json:"address"`
	}

	// Parse body JSON
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	// Mapping ke model
	parentBank := models.ParentBank{
		District: body.District,
		Province: body.Province,
		Address:  body.Address,
	}

	// Simpan ke database
	if err := configs.DB.Create(&parentBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to create parent bank", nil, nil)
	}

	return helpers.Response(c, 201, "Success", "Parent Bank created successfully", parentBank, nil)
}

// Function for Mobile in Parent Bank
func GetParentBankID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, 400, "failed", "Invalid user ID", nil, nil)
	}

	var user models.User
	if err := configs.DB.Preload("ParentBank").Preload("Role").Where("id = ?", id).First(&user).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Parent Bank not found", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Data Found", user.ParentBank, nil)
}

// Function for Admin in Web (Update Parent Bank)
func UpdateParentBank(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		District string `json:"district"`
		Province string `json:"province"`
		Address  string `json:"address"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	var parentBank models.ParentBank
	if err := configs.DB.First(&parentBank, id).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Parent Bank not found", nil, nil)
	}

	parentBank.District = body.District
	parentBank.Province = body.Province
	parentBank.Address = body.Address

	if err := configs.DB.Save(&parentBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to update parent bank", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Parent Bank updated successfully", parentBank, nil)
}

// Function for Admin in Web (Delete Parent Bank)
func DeleteParentBank(c *fiber.Ctx) error {
	id := c.Params("id")

	var parentBank models.ParentBank
	if err := configs.DB.First(&parentBank, id).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Parent Bank not found", nil, nil)
	}

	if err := configs.DB.Delete(&parentBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to delete parent bank", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Parent Bank deleted successfully", nil, nil)
}
