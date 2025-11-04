package models

import (
	"time"

	"gorm.io/gorm"
)

type Marketing struct {
	Id          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Image       string         `json:"image" gorm:"type:varchar(255)"`
	Title       string         `json:"title" gorm:"type:varchar(255);"`    // Judul Promosi
	StartDate   time.Time      `json:"start_date"`                         // Tanggal & Waktu Mulai
	EndDate     time.Time      `json:"end_date"`                           // Tanggal & Waktu Berakhir
	Broadcast   string         `json:"broadcast" gorm:"type:varchar(100)"` // Fifth broadcast, dll
	Description string         `json:"description" gorm:"type:text;"`      // Deskripsi
}
