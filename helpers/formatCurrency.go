package helpers

import (
	"fmt"
	"strings"
)

// Helper function untuk format currency
func FormatCurrency(amount int) string {
	// Konversi ke float untuk formatting
	floatAmount := float64(amount)

	// Format dengan separator ribuan dan 2 digit desimal
	str := fmt.Sprintf("Rp %.2f", floatAmount)

	// Tambahkan separator ribuan
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = "." + parts[1]
	}

	// Format ribuan (skip "Rp " di depan)
	rpPrefix := "Rp "
	numberPart := integerPart[len(rpPrefix):]

	var formattedInteger string
	count := 0
	// Balik string untuk memudahkan penambahan titik
	for i := len(numberPart) - 1; i >= 0; i-- {
		if count > 0 && count%3 == 0 {
			formattedInteger = "." + formattedInteger
		}
		formattedInteger = string(numberPart[i]) + formattedInteger
		count++
	}

	return rpPrefix + formattedInteger + decimalPart
}