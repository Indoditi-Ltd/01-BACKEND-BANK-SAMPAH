package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
)

func SeedDivision() error {
	divisions := []models.Division{
		{Name: "Finance"},
		{Name: "IT"},
		{Name: "Manager"},
		{Name: "HR Mitra"},
	}

	for _, division := range divisions {
		var existingdivision models.Division
		if err := configs.DB.Where("name = ?", division.Name).First(&existingdivision).Error; err == nil {
			continue
		}
		if err := configs.DB.Create(&division).Error; err != nil {
			return err
		}
	}

	return nil
}
