package configs

import "backend-mulungs/models"

func DatabaseSync() {
	DB.AutoMigrate(&models.User{}, &models.Transaction{})
}
