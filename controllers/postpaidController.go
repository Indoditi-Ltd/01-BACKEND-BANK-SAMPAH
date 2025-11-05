package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	// Cek jika response ada data
	if apiResponse["data"] == nil {
		return helpers.Response(c, 404, "Not Found", "Tidak ada data ditemukan", nil, nil)
	}

	// Extract data pasca dari response
	data, ok := apiResponse["data"].(map[string]interface{})
	if !ok {
		return helpers.Response(c, 400, "Failed", "Format data tidak valid", nil, nil)
	}

	pasca, ok := data["pasca"].([]interface{})
	if !ok {
		return helpers.Response(c, 404, "Not Found", "Tidak ada data pasca ditemukan", nil, nil)
	}

	// --- Filter data ---
	var filtered []map[string]interface{}
	
	for _, item := range pasca {
		itemMap, ok := item.(map[string]interface{})
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
					filtered = append(filtered, map[string]interface{}{
						"code": itemCode,
						"name": itemName,
						"type": itemType,
					})
				} else if bpjsType == "ketenagakerjaan" && isBPJSKetenagakerjaan(itemMap) {
					filtered = append(filtered, map[string]interface{}{
						"code": itemCode,
						"name": itemName,
						"type": itemType,
					})
				}
			} else {
				// Jika tidak ada filter bpjs_type, atau bukan type bpjs, tampilkan semua
				filtered = append(filtered, map[string]interface{}{
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
	var reqBody models.ExternalPaymentRequest

	if err := c.BodyParser(&reqBody); err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid request body", nil, nil)
	}

	username := os.Getenv("IDENTITY")
	sign := helpers.MakeSignPricelist(reqBody.TrID) // buat tanda tangan sesuai format API kamu
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
		return helpers.Response(c, 400, "Failed", "Failed decode response", nil, nil)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed decode response", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Success Inquiry", result["data"], nil)
}
