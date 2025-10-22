package models

import (
	"time"

	"gorm.io/gorm"
)

type Company struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Balance   int            `json:"balance" gorm:"int(255)"`
}
