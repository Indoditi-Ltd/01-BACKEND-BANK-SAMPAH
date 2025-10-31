package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"github.com/gofiber/fiber/v2"
)



func CreateMargin(c *fiber.Ctx) error {
	var body struct {
		Margin int `json:"margin"`
	}

	// parsing body JSON
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to read body", nil, nil)
	}

	// cek apakah sudah ada data margin di tabel
	var existing models.Ppob
	if err := configs.DB.First(&existing).Error; err == nil {
		// jika sudah ada → update
		existing.Margin = body.Margin
		if err := configs.DB.Save(&existing).Error; err != nil {
			return helpers.Response(c, 400, "Failed", "Failed to update margin", nil, nil)
		}
		return helpers.Response(c, 200, "Success", "Margin updated successfully", existing, nil)
	}

	// kalau belum ada → buat baru
	ppob := models.Ppob{
		Margin: body.Margin,
	}
	if err := configs.DB.Create(&ppob).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to create margin", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Margin created successfully", ppob, nil)
}
