package models

import (
	"time"

	"gorm.io/gorm"
)

type WasteDeposit struct {
	Id          uint               `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `json:"deleted_at" gorm:"index"`
	UserID      uint               `json:"-"`
	User        User               `json:"user" gorm:"foreignKey:UserID"`
	TotalWeight float64            `json:"total_weight" gorm:"type:decimal(10,2);not null"`
	TotalPrice  int                `json:"total_price" gorm:"type:int;not null"`
	ReferenceID string             `json:"reference_id" gorm:"type:varchar(100);uniqueIndex"`
	Items       []WasteDepositItem `json:"items" gorm:"foreignKey:WasteDepositID"`
}
