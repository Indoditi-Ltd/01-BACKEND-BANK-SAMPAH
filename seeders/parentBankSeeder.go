package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
)

func SeedParentBank() error {
	parentBanks := []models.ParentBank{
		{District: "Kabupaten Jember", Province: "Jawa Timur", Address: "Jl. Jawa No. 50, Jember, Jawa Timur"},
		{District: "Kabupaten Lumajang", Province: "Jawa Timur", Address: "Jl. Subandi No. 50, Lumajang, Jawa Timur"},
	}

	for _, parentBank := range parentBanks {
		var existingParentBank models.ParentBank
		if err := configs.DB.Where("address = ?", parentBank.Address).First(&existingParentBank).Error; err == nil {
			continue
		}
		if err := configs.DB.Create(&parentBank).Error; err != nil {
			return err
		}
	}
	return nil
}
