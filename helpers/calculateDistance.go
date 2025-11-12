package helpers

import "math"

// calculateDistance - Calculate distance between two coordinates menggunakan Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + 
		 math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		 math.Sin(dlon/2)*math.Sin(dlon/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Earth radius in kilometers
	radius := 6371.0

	return radius * c
}