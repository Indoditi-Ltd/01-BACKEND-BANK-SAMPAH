package models

import (
	"time"

	"gorm.io/gorm"
)

type Ppob struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Margin    int            `json:"margin" gorm:"type:double;not null"`
}

// Struktur body yang dikirim ke API eksternal
type ExternalRequestPrepaid struct {
	Status   string `json:"status"`
	Username string `json:"username"`
	Sign     string `json:"sign"`
}

// Struktur response dari API eksternal
type PrepaidResponse struct {
	Data struct {
		Pricelist []ProductPrepaid `json:"pricelist"`
		RC        string           `json:"rc"`
		Message   string           `json:"message"`
	} `json:"data"`
}

// struktur produk response
type ProductPrepaid struct {
	ProductCode        string  `json:"product_code"`
	ProductDescription string  `json:"product_description"`
	ProductNominal     string  `json:"product_nominal"`
	ProductDetails     string  `json:"product_details"`
	ProductPrice       float64 `json:"product_price"`
	ProductType        string  `json:"product_type"`
	ActivePeriod       string  `json:"active_period"`
	Status             string  `json:"status"`
	IconURL            string  `json:"icon_url"`
	ProductCategory    string  `json:"product_category"`
}

// Struktur body yang dikirim ke API eksternal
type ExternalRequestPostpaid struct {
	Commands string  `json:"commands"`
	Status   string  `json:"status"`
	Username string  `json:"username"`
	Sign     string  `json:"sign"`
	Province *string `json:"province,omitempty"`
}

// response response postpaid
type PostpaidResponse struct {
	Data struct {
		Pasca []ProductPostpaid `json:"pasca"`
	} `json:"data"`
}

// response response produk postpaid
type ProductPostpaid struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Status   int     `json:"status"`
	Fee      float64 `json:"fee"`
	Komisi   int     `json:"komisi"`
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Province string  `json:"province"`
}

type ExternalRequestTopup struct {
	Username    string `json:"username"`
	RefId       string `json:"ref_id"`
	CustomerId  string `json:"customer_id"`
	ProductCode string `json:"product_code"`
	Sign        string `json:"sign"`
}

type PrepaidResponseTopup struct {
	Data DataTopup `json:"data"`
}

type DataTopup struct {
	RefId       string  `json:"ref_id"`
	Status      int     `json:"status"`
	ProductCode string  `json:"product_code"`
	CustomerId  string  `json:"customer_id"`
	Price       float64 `json:"price"`
	Message     string  `json:"message"`
	Balance     float64 `json:"balance"`
	TrId        float64 `json:"tr_id"`
	Rc          string  `json:"rc"`
}
