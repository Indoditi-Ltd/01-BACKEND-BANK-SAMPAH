package models

import (
	"time"

	"gorm.io/gorm"
)

type PickupRequest struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// User yang request penjemputan
	UserID uint `json:"-"`
	User   User `json:"user" gorm:"foreignKey:UserID"`

	// Bank yang dituju
	ChildBankID *uint      `json:"-"`
	ChildBank   *ChildBank `json:"child_bank" gorm:"foreignKey:ChildBankID"`
	ParentBankID *uint      `json:"-"`
	ParentBank   *ParentBank `json:"parent_bank" gorm:"foreignKey:ParentBankID"`

	// Data sederhana - hanya koordinat dan status
	Latitude  float64 `json:"latitude" gorm:"type:decimal(10,8);not null"`
	Longitude float64 `json:"longitude" gorm:"type:decimal(11,8);not null"`
	Status    string  `json:"status" gorm:"type:enum('pending','accept','complete','reject');default:'pending'"`
}