package models

import (
	"time"

	"gorm.io/gorm"
)

type DonationHistory struct {
	Id           uint           `json:"id" gorm:"primarykey"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	UserID       uint           `json:"-"`
	User         User           `json:"user" gorm:"foreignKey:UserID"`
	DonationID   uint           `json:"-"`
	Donation     Donation       `json:"donation" gorm:"foreignKey:DonationID"`
	Amount       int            `json:"amount" gorm:"type:int;not null"`
	// Tidak perlu field Action karena langsung donation saja
}