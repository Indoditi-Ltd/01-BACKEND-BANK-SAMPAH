package models

type RegionResponse struct {
	Id        string  `json:"id"`
	Name      string  `json:"nama"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
