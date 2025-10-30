package models

import (
	"time"

	"gorm.io/gorm"
)

type HistoryModel struct {
	Id            uint           `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	UserID        uint           `json:"-"`
	User          User           `json:"user" gorm:"foreignKey:UserID"`
	RefID         string         `json:"ref_id" gorm:"index"`
	ProductName   string         `json:"product_name" gorm:"type:varchar(255)"`
	ProductPrice  string          `json:"product_price"`
	ProductType   string         `json:"product_type"`
	UserNumber    string         `json:"user_number"`
	TotalPrice    string          `json:"total_price"`
	StroomToken   string         `json:"stroom_token"`
	BillingPeriod string         `json:"billing_period"`
	Year          string         `json:"year"`
	Province      string         `json:"province"`
	Region        string         `json:"region"`
	Status        string         `json:"status"`
}
