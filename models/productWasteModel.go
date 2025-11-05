package models

import (
	"time"

	"gorm.io/gorm"
)

type ProductWaste struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	WasteType string         `json:"waste_type" gorm:"type:varchar(100);not null"`              // Jenis Sampah
	Image     string         `json:"image" gorm:"type:varchar(255)"`                            // Gambar
	Unit      string         `json:"unit" gorm:"type:varchar(50);not null"`                     // Satuan (kg, liter, etc)
	Price     int            `json:"price" gorm:"type:int;not null"`                            // Harga
	Category  string         `json:"category" gorm:"type:enum('organik','anorganik');not null"` // Kategori
	// Weight    float64        `json:"weight" gorm:"type:decimal(10,2)"`                          // Berat dalam kg
}
