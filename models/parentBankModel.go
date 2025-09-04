package models

import (
	"time"

	"gorm.io/gorm"
)

type ParentBank struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	District  string         `json:"district" gorm:"type:varchar(100);not null"`
	Province  string         `json:"province" gorm:"type:varchar(100);not null"`
	Address   string         `json:"address" gorm:"type:text;not null"`
}
