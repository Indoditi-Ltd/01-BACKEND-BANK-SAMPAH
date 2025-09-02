package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	Id         uint           `json:"id" gorm:"primarykey"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Name string `json:"name" gorm:"type:varchar(50);unique;not null"`
}