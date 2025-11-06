package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)
func GetListPostpaid(c *fiber.Ctx) error {
	typ := c.Params("type")
	province := c.Query("province")
	bpjsType := c.Query("bpjs_type") // Query parameter baru: "kesehatan" atau "ketenagakerjaan"
	username := os.Getenv("IDENTITY")
	sign := helpers.MakeSignPricelist("pl")

	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}

	var settings models.Ppob
	if err := configs.DB.First(&settings).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal mengambil margin dari database", nil, nil)
	}

	reqBody := models.ExternalRequestPostpaid{
		Commands: "pricelist-pasca",
		Status:   "all",
		Username: username,
		Sign:     sign,
	}

	if typ == "pdam" && province != "" {
		reqBody.Province = &province
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal encode request body", nil, nil)
	}

	url := fmt.Sprintf("https://testpostpaid.mobilepulsa.net/api/v1/bill/check/%s", typ)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Failed request API external", nil, nil)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal membaca response body", nil, nil)
	}

	// Decode response ke map[string]interface{} untuk handle struktur dinamis
	var apiResponse map[string]any
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	// Cek jika response ada data
	if apiResponse["data"] == nil {
		return helpers.Response(c, 404, "Not Found", "Tidak ada data ditemukan", nil, nil)
	}

	// Extract data pasca dari response
	data, ok := apiResponse["data"].(map[string]any)
	if !ok {
		return helpers.Response(c, 400, "Failed", "Format data tidak valid", nil, nil)
	}

	pasca, ok := data["pasca"].([]any)
	if !ok {
		return helpers.Response(c, 404, "Not Found", "Tidak ada data pasca ditemukan", nil, nil)
	}

	// --- Filter data ---
	var filtered []map[string]any
	
	for _, item := range pasca {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		itemType, typeOk := itemMap["type"].(string)
		itemCode, codeOk := itemMap["code"].(string)
		itemName, nameOk := itemMap["name"].(string)

		if !typeOk || !codeOk || !nameOk {
			continue
		}

		// Filter berdasarkan type utama (bpjs, pdam, dll)
		if typ == "" || strings.EqualFold(itemType, typ) {
			// Jika query bpjs_type ada dan type adalah bpjs, filter lebih lanjut
			if bpjsType != "" && strings.EqualFold(itemType, "bpjs") {
				if bpjsType == "kesehatan" && isBPJSKesehatan(itemMap) {
					filtered = append(filtered, map[string]any{
						"code": itemCode,
						"name": itemName,
						"type": itemType,
					})
				} else if bpjsType == "ketenagakerjaan" && isBPJSKetenagakerjaan(itemMap) {
					filtered = append(filtered, map[string]any{
						"code": itemCode,
						"name": itemName,
						"type": itemType,
					})
				}
			} else {
				// Jika tidak ada filter bpjs_type, atau bukan type bpjs, tampilkan semua
				filtered = append(filtered, map[string]any{
					"code": itemCode,
					"name": itemName,
					"type": itemType,
				})
			}
		}
	}

	if len(filtered) == 0 {
		message := "Tidak ada data ditemukan untuk tipe tersebut"
		if bpjsType != "" {
			message = fmt.Sprintf("Tidak ada data BPJS %s ditemukan", bpjsType)
		}
		return helpers.Response(c, 404, "Not Found", message, nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", filtered, nil)
}

// Helper function untuk mendeteksi BPJS Kesehatan
func isBPJSKesehatan(item map[string]interface{}) bool {
	bpjsKesehatanCodes := []string{"BPJS", "BPJSB"}
	code, ok := item["code"].(string)
	if !ok {
		return false
	}
	
	for _, kesehatanCode := range bpjsKesehatanCodes {
		if code == kesehatanCode {
			return true
		}
	}
	return false
}

// Helper function untuk BPJS Ketenagakerjaan
func isBPJSKetenagakerjaan(item map[string]interface{}) bool {
	bpjsTKCodes := []string{"BPJSTK", "BPJSTKPU", "BPJSTKPUREG", "BPJSTKREG"}
	code, ok := item["code"].(string)
	if !ok {
		return false
	}
	
	for _, tkCode := range bpjsTKCodes {
		if code == tkCode {
			return true
		}
	}
	return false
}

func PostpaidInquiry(c *fiber.Ctx) error {
	// Ambil data request dari body
	var reqBody models.ExternalInquiryRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid request body", nil, nil)
	}

	username := os.Getenv("IDENTITY")
	sign := helpers.MakeSignPricelist(reqBody.RefID) // buat tanda tangan sesuai format API kamu

	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}

	// Siapkan body request ke IAK
	payload := map[string]any{
		"commands": "inq-pasca",
		"username": username,
		"code":     reqBody.Code,
		"hp":       reqBody.Hp,
		"ref_id":   reqBody.RefID,
		"month":    reqBody.Month,
		"sign":     sign,
	}

	jsonBody, _ := json.Marshal(payload)

	// Kirim request ke API eksternal
	url := "https://testpostpaid.mobilepulsa.net/api/v1/bill/check"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Failed request API external", nil, nil)
	}
	defer resp.Body.Close()

	// Baca respons dari eksternal API
	body, _ := io.ReadAll(resp.Body)

	// Unmarshal hasil API
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed decode response", nil, nil)
	}

	// Return hasil inquiry dari API langsung ke client
	return helpers.Response(c, 200, "Success", "Success Inquiry", result["data"], nil)
}
func PaymentPostpaid(c *fiber.Ctx) error {
	var reqBody struct {
		TrID   string `json:"tr_id"`
		UserID string `json:"user_id"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid request body", nil, nil)
	}

	// Convert user_id dari string ke uint
	userID, err := strconv.ParseUint(reqBody.UserID, 10, 32)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid user ID", nil, nil)
	}

	username := os.Getenv("IDENTITY")
	sign := helpers.MakeSignPricelist(reqBody.TrID)
	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}

	// Siapkan body request ke IAK
	payload := map[string]any{
		"commands": "pay-pasca",
		"username": username,
		"tr_id":    reqBody.TrID,
		"sign":     sign,
	}
	jsonBody, _ := json.Marshal(payload)
	url := "https://testpostpaid.mobilepulsa.net/api/v1/bill/check"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to call external API", nil, nil)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed decode response", nil, nil)
	}

	// Simpan ke history jika response success
	if data, ok := result["data"].(map[string]interface{}); ok {
		// Gunakan goroutine agar tidak blocking response ke client
		go func() {
			if err := savePostpaidHistory(uint(userID), data); err != nil {
				log.Printf("Failed to save payment history: %v", err)
			} else {
				log.Printf("Successfully saved payment history for user %d", userID)
			}
		}()
	}

	return helpers.Response(c, 200, "Success", "Success Payment", result["data"], nil)
}

// Fungsi untuk menentukan product type berdasarkan response data
func determineProductType(data map[string]interface{}) string {
	if code, ok := data["code"].(string); ok {
		// List provider internet
		internetProviders := []string{"CBN", "FIRSTMEDIA", "MYREPUBLIC", "SPEEDY", "SPEEDYB", "TELKOMPSTN"}
		
		// Cek apakah code termasuk provider internet
		for _, provider := range internetProviders {
			if code == provider {
				return "internet"
			}
		}
		
		// Jenis layanan lainnya
		switch {
		case strings.HasPrefix(code, "BPJS") && code != "BPJSTK" && code != "BPJSTKPU":
			return "bpjs_health"
		case code == "BPJSTK" || code == "BPJSTKPU":
			return "bpjs_employment"
		case strings.HasPrefix(code, "PBB"):
			return "pbb"
		case strings.HasPrefix(code, "PDAM"):
			return "pdam"
		case code == "PLNPOSTPAID":
			return "pln_postpaid"
		case code == "PGAS":
			return "gas"
		default:
			return "other_postpaid"
		}
	}
	return "unknown"
}

// Helper function untuk generate product name
func getProductName(data map[string]interface{}) string {
	productType := determineProductType(data)
	
	if trName, ok := data["tr_name"].(string); ok {
		switch productType {
		case "bpjs_health":
			return fmt.Sprintf("BPJS Kesehatan - %s", trName)
		case "bpjs_employment":
			return fmt.Sprintf("BPJS Ketenagakerjaan - %s", trName)
		case "bpjs_employment_pu":
			return fmt.Sprintf("BPJS Ketenagakerjaan PU - %s", trName)
		case "pbb":
			return fmt.Sprintf("Pajak Bumi Bangunan - %s", trName)
		case "pdam":
			return fmt.Sprintf("PDAM - %s", trName)
		case "pln_postpaid":
			return fmt.Sprintf("PLN Pascabayar - %s", trName)
		case "telkom":
			return fmt.Sprintf("Telkom - %s", trName)
		case "internet":
			return fmt.Sprintf("Internet - %s", trName)
		case "gas":
			return fmt.Sprintf("Gas - %s", trName)
		default:
			return fmt.Sprintf("Layanan Postpaid - %s", trName)
		}
	}
	
	// Fallback jika tidak ada tr_name
	switch productType {
	case "bpjs_health":
		return "BPJS Kesehatan"
	case "bpjs_employment":
		return "BPJS Ketenagakerjaan"
	case "pbb":
		return "Pajak Bumi Bangunan"
	case "pdam":
		return "PDAM"
	case "pln_postpaid":
		return "PLN Pascabayar"
	case "telkom":
		return "Telkom"
	case "internet":
		return "Internet"
	case "gas":
		return "Gas"
	default:
		return "Layanan Postpaid"
	}
}

// Fungsi untuk menyimpan history postpaid
func savePostpaidHistory(userID uint, data map[string]interface{}) error {
	history := models.HistoryModel{
		UserID:      userID,
		ProductType: determineProductType(data),
		Status:      "success",
	}

	// Extract dan mapping data dari response IAK
	if trID, ok := data["tr_id"].(float64); ok {
		history.RefID = fmt.Sprintf("%.0f", trID)
	} else if refID, ok := data["ref_id"].(string); ok {
		history.RefID = refID
	}

	if hp, ok := data["hp"].(string); ok {
		history.UserNumber = hp
	}

	// Set product name
	history.ProductName = getProductName(data)

	if period, ok := data["period"].(string); ok {
		history.BillingPeriod = period
	}

	// Format amount dengan currency
	if nominal, ok := data["nominal"].(float64); ok {
		history.ProductPrice = fmt.Sprintf("Rp. %.0f", nominal)
	}
	if price, ok := data["price"].(float64); ok {
		history.TotalPrice = fmt.Sprintf("Rp. %.0f", price)
	}

	// Simpan desc hanya jika ada dan berisi data
	if desc, ok := data["desc"].(map[string]interface{}); ok && len(desc) > 0 {
		descJSON, _ := json.Marshal(desc)
		history.Province = string(descJSON)
	}

	// Set status berdasarkan response code
	if responseCode, ok := data["response_code"].(string); ok {
		if responseCode != "00" {
			history.Status = "failed"
		}
	}

	// Year, Region, StroomToken dibiarkan kosong saja

	return configs.DB.Create(&history).Error
}