package models

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	Name string `json:"name" gorm:"type:varchar(50);unique;not null"`
}