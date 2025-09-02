package models

import "gorm.io/gorm"

type Transaction struct {
	gorm.Model
	UserID  uint `json:"user_id" gorm:"not null"`
	User    User `gorm:"foreignkey:UserID"`
	Balance int  `json:"balance" gorm:"not null"`
	Status  bool `json:"confirm" gorm:"type:enum('pending', 'confirm', 'reject')"`
	AdminID uint `json:"admin_id"`
	Admin   User `gorm:"foreignkey:AdminID"`
}
