package models

import (
	"time"

	"gorm.io/gorm"
)

type ChildBank struct {
	Id           uint           `json:"id" gorm:"primarykey"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Subdistrict  string         `json:"subdistrict" gorm:"type:varchar(100);not null"`
	RT           string         `json:"rt" gorm:"type:varchar(100);not null"`
	RW           string         `json:"rw" gorm:"type:varchar(100);not null"`
	Address      string         `json:"address" gorm:"type:text;not null"`
	ParentBankID uint           `json:"-"`
	ParentBank   ParentBank     `json:"parent_bank" gorm:"foreignKey:ParentBankID"`
}

type ChildBankResponse struct {
	Id           uint   `json:"id"`
	Subdistrict  string `json:"subdistrict"`
	RT           string `json:"rt"`
	RW           string `json:"rw"`
	Address      string `json:"address"`
	ParentBankID uint   `json:"parent_bank_id"`
}
