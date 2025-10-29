package configs

import "backend-mulungs/models"

func DatabaseSync() {
	DB.AutoMigrate(&models.User{}, &models.Transaction{}, &models.ParentBank{}, &models.ChildBank{}, &models.Ppob{}, &models.Company{}, &models.HistoryModel{})
}
