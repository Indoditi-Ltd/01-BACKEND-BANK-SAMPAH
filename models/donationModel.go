package models

import (
	"time"

	"gorm.io/gorm"
)

type Donation struct {
	Id          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Image       string         `json:"image" gorm:"type:varchar(255)"`
	StartDate   time.Time      `json:"start_date" gorm:"not null"`
	EndDate     time.Time      `json:"end_date" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text;not null"`
	Status      string         `json:"status" gorm:"type:enum('pending','completed','cancelled');default:'pending'"`
	// Hapus field title dan target amountj
	// HANYA TAMBAH INI SAJA - untuk menampung total dana terkumpul
	CurrentAmount int `json:"current_amount" gorm:"type:int;default:0"`
	// Histories     []DonationHistory `json:"histories" gorm:"foreignKey:DonationID"` // ✅ Relasi ke riwayat
}
