package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
)

func SeedPlan() error {
	plans := []models.Plan{
		{Name: "Platinum"},
		{Name: "Gold"},
	}

	for _, plan := range plans {
		var existingPlan models.Plan
		if err := configs.DB.Where("name = ?", plan.Name).First(&existingPlan).Error; err == nil {
			continue
		}
		if err := configs.DB.Create(&plan).Error; err != nil {
			return err
		}
	}

	return nil
}
