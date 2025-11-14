package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/models"
	"log"

	"gorm.io/gorm"
)

func SeedChildBanks() error {
	log.Println("🌱 Seeding Child Bank data...")

	db := configs.DB

	// Cek apakah data child bank sudah ada
	var childBanks []models.ChildBank
	result := db.Find(&childBanks)

	if result.Error != nil && result.Error == gorm.ErrRecordNotFound || len(childBanks) == 0 {
		// Buat data child bank baru
		childBankData := []models.ChildBank{
			{
				Subdistrict:  "Kecamatan Jember",
				RT:           "001",
				RW:           "002",
				Address:      "Jl. Merdeka No. 123, Kecamatan Jember, Jawa Timur",
				Latitude:     -8.1844859,
				Longitude:    113.6680757,
				ParentBankID: 1, // Sesuaikan dengan ID parent bank yang ada
				Norek:        1001001,
				Balance:      5000000,
			},
			{
				Subdistrict:  "Kecamatan Kaliwates",
				RT:           "003",
				RW:           "004",
				Address:      "Jl. Gajah Mada No. 45, Kecamatan Kaliwates, Jawa Timur",
				Latitude:     -8.1723578,
				Longitude:    113.6994243,
				ParentBankID: 1, // Sesuaikan dengan ID parent bank yang ada
				Norek:        1001002,
				Balance:      7500000,
			},
		}

		if err := db.Create(&childBankData).Error; err != nil {
			log.Printf("❌ Error seeding child banks: %v", err)
			return err
		}

		log.Printf("✅ Child Bank seeder executed successfully - %d child banks created", len(childBankData))
	} else if result.Error != nil {
		log.Printf("❌ Error checking child banks: %v", result.Error)
		return result.Error
	} else {
		log.Printf("⏩ %d Child Banks already exist, skipping seeder", len(childBanks))
	}

	return nil
}