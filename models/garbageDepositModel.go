package models

import (
	"time"

	"gorm.io/gorm"
)

type GerbageDeposit struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	UserID    uint           `json:"-"`
	User      User           `json:"user" gorm:"foreignKey:UserID"`
	
}
