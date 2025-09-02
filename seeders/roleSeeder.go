package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
)

func SeedRole() error {
	roles := []models.Role{
		{Name: "admin"},
		{Name: "user"},
		{Name: "parent bank"},
		{Name: "child bank"},
		{Name: "partner"},
	}

	for _, role := range roles {
		var existingRole models.Role
		if err := configs.DB.Where("name = ?", role.Name).First(&existingRole).Error; err == nil {
			continue // Lewati jika role sudah ada
		}
		if err := configs.DB.Create(&role).Error; err != nil {
			return err
		}
	}
	return nil
}
