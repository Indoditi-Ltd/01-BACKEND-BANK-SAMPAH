package configs

import "backend-mulungs/models"

func DatabaseSync() {
	DB.AutoMigrate(
		&models.Transaction{},
		&models.ParentBank{},
		&models.ChildBank{},
		&models.User{},
		&models.Ppob{},
		&models.Company{},
		&models.HistoryModel{},
		&models.Marketing{},
		&models.ProductWaste{},
		&models.Donation{},
		&models.DonationHistory{},
		&models.WasteDepositItem{},
		&models.WasteDeposit{},
		&models.PickupRequest{},
	)
}
