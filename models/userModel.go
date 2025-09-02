package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name       string    `json:"name" gorm:"type:varchar(100);not null"`
	Email      string    `json:"email" gorm:"type:varchar(100);unique;not null"`
	Password   string    `json:"-" gorm:"type:varchar(255);not null"`
	Phone      string    `json:"phone" gorm:"type:varchar(20)"`
	Address    string    `json:"address" gorm:"type:varchar(255)"`
	Photo      string    `json:"photo" gorm:"type:varchar(255)"`
	DivisionID *uint     `json:"-"`
	Division   *Division `gorm:"foreignKey:DivisionID"`
	RoleID     uint      `json:"-"`
	Role       Role      `gorm:"foreignKey:RoleID"`
	PlanID     *uint     `json:"-"`
	Plan       *Plan     `gorm:"foreignKey:PlanID"`
}
