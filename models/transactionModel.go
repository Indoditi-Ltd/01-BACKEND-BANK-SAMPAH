package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	UserID    uint           `json:"-" gorm:"not null"`
	User      User           `json:"data_user" gorm:"foreignkey:UserID"`
	Balance   int            `json:"balance" gorm:"not null"`
	Type      string         `json:"type" gorm:"type:enum('topup', 'withdraw')"`
	Status    string         `json:"status" gorm:"type:enum('pending', 'confirm', 'reject')"`
	Desc      string         `json:"desc" grom:"text"`
	AdminID   *uint          `json:"-"`
	Admin     *User          `json:"data_admin" gorm:"foreignkey:AdminID"`
}


// Untuk response setelah CREATE (ringkas)
type TransactionCreateResponse struct {
    ID      uint   `json:"id"`
    UserID  uint   `json:"user_id"`
    Balance int    `json:"balance"`
    Type    string `json:"type"`
    Status  string `json:"status"`
    Desc    string `json:"description"`
}
