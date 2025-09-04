package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	Id           uint           `json:"id" gorm:"primarykey"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Name         string         `json:"name" gorm:"type:varchar(100);not null"`
	Email        string         `json:"email" gorm:"type:varchar(100);unique;not null"`
	Password     string         `json:"-" gorm:"type:varchar(255);not null"`
	Phone        string         `json:"phone" gorm:"type:varchar(20)"`
	Address      string         `json:"address" gorm:"type:varchar(255)"`
	Photo        string         `json:"photo" gorm:"type:varchar(255)"`
	DivisionID   *uint          `json:"-"`
	Division     *Division      `json:"division" gorm:"foreignKey:DivisionID"`
	RoleID       uint           `json:"-"`
	Role         Role           `json:"role" gorm:"foreignKey:RoleID"`
	PlanID       *uint          `json:"-"`
	Plan         *Plan          `json:"plan" gorm:"foreignKey:PlanID"`
	ParentBankID *uint          `json:"-"`
	ParentBank   *ParentBank    `json:"parent_bank" gorm:"foreignKey:ParentBankID"`
}
