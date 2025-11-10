package models

import (
	"time"

	"gorm.io/gorm"
)

type WasteDepositItem struct {
	Id             uint           `json:"id" gorm:"primarykey"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	WasteDepositID uint           `json:"-"`
	WasteDeposit   WasteDeposit   `json:"-" gorm:"foreignKey:WasteDepositID"`
	ProductWasteID uint           `json:"-"`
	ProductWaste   ProductWaste   `json:"product_waste" gorm:"foreignKey:ProductWasteID"`
	Category       string         `json:"category" gorm:"type:enum('organik','anorganik');not null"`
	Weight         float64        `json:"weight" gorm:"type:decimal(10,2);not null"`
	Unit           string         `json:"unit" gorm:"type:varchar(20);not null"` // kg, ons, etc
	SubTotal       int            `json:"sub_total" gorm:"type:int;not null"`
	Photo          string         `json:"photo" gorm:"type:varchar(255)"`
}
