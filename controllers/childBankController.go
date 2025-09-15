package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CREATE
// CREATE
func CreateChildBank(c *fiber.Ctx) error {
	// bikin struct body langsung di dalam function
	var body struct {
		Subdistrict  string `json:"subdistrict"`
		RT           string `json:"rt"`
		RW           string `json:"rw"`
		Address      string `json:"address"`
		ParentBankID uint   `json:"parentBank_id"`
		Norek        uint   `json:"norek"`
	}

	// Parse body JSON
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	// mapping ke model ChildBank
	childBank := models.ChildBank{
		Subdistrict:  body.Subdistrict,
		RT:           body.RT,
		RW:           body.RW,
		Address:      body.Address,
		ParentBankID: body.ParentBankID,
		Norek:        body.Norek,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// simpan ke database
	if err := configs.DB.Create(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	res := models.ChildBankResponse{
		Id:           childBank.Id,
		Subdistrict:  childBank.Subdistrict,
		RT:           childBank.RT,
		RW:           childBank.RW,
		Address:      childBank.Address,
		ParentBankID: childBank.ParentBankID,
		Norek:        childBank.Norek,
	}

	return helpers.Response(c, 201, "Success", "Child Bank created successfully", res, nil)
}

// READ ALL
func GetAllChildBanks(c *fiber.Ctx) error {
	var childBanks []models.ChildBank

	if err := configs.DB.Preload("ParentBank").Find(&childBanks).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Child Banks retrieved successfully", childBanks, nil)
}

// READ BY ID
func GetChildBankById(c *fiber.Ctx) error {
	id := c.Params("id")
	var childBank models.ChildBank

	if err := configs.DB.Preload("ParentBank").First(&childBank, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Child Bank retrieved successfully", childBank, nil)
}

// Function for Admin in Web (Update Child Bank)
func UpdateChildBank(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Subdistrict  string `json:"subdistrict"`
		RT           string `json:"rt"`
		RW           string `json:"rw"`
		Address      string `json:"address"`
		ParentBankID uint   `json:"parentBank_id"`
		Norek        uint   `json:"norek"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	var childBank models.ChildBank
	if err := configs.DB.First(&childBank, id).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
	}

	childBank.Subdistrict = body.Subdistrict
	childBank.RT = body.RT
	childBank.RW = body.RW
	childBank.Address = body.Address
	childBank.ParentBankID = body.ParentBankID
	childBank.Norek = body.Norek

	if err := configs.DB.Save(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to update child bank", nil, nil)
	}

	res := models.ChildBankResponse{
		Id:           childBank.Id,
		Subdistrict:  childBank.Subdistrict,
		RT:           childBank.RT,
		RW:           childBank.RW,
		Address:      childBank.Address,
		ParentBankID: childBank.ParentBankID,
		Norek:        childBank.Norek,
	}

	return helpers.Response(c, 200, "Success", "Child Bank updated successfully", res, nil)
}

// DELETE
func DeleteChildBank(c *fiber.Ctx) error {
	id := c.Params("id")
	var childBank models.ChildBank

	if err := configs.DB.First(&childBank, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	if err := configs.DB.Delete(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Child Bank deleted successfully", nil, nil)
}
