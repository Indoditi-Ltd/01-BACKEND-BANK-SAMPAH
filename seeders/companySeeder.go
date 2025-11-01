package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
	"log"

	"gorm.io/gorm"
)

func SeedCompany() error {
	log.Println("🌱 Seeding Company data...")

	db := configs.DB

	// Cek apakah data company sudah ada
	var company models.Company
	result := db.First(&company)

	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		// Buat data company baru dengan balance 0
		companyData := models.Company{
			Balance: 0,
		}

		if err := db.Create(&companyData).Error; err != nil {
			log.Printf("❌ Error seeding company: %v", err)
			return err
		}

		log.Println("✅ Company seeder executed successfully - Company created with balance 0")
	} else if result.Error != nil {
		log.Printf("❌ Error checking company: %v", result.Error)
		return result.Error
	} else {
		log.Println("⏩ Company already exists, skipping seeder")
	}

	return nil
}
