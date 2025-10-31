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
	username := os.Getenv("IDENTITY")
	sign := helpers.MakeSignPricelist("pl")

	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}

	var settings models.Ppob
	if err := configs.DB.First(&settings).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal mengambil margin dari database", nil, nil)
	}
	// margin := float64(settings.Margin)

	reqBody := models.ExternalRequestPostpaid{
		Commands: "pricelist-pasca",
		Status:   "all",
		Username: username,
		Sign:     sign,
	}

	if typ == "pdam" && province != "" {
		reqBody.Province = &province
	}

	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("https://testpostpaid.mobilepulsa.net/api/v1/bill/check/%s", typ)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Failed request API external", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result models.PostpaidResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	// --- Filter dan ambil field tertentu ---
	var filtered []map[string]interface{}
	for _, item := range result.Data.Pasca {
		if typ == "" || strings.EqualFold(item.Type, typ) { // case-insensitive
			filtered = append(filtered, map[string]interface{}{
				"code": item.Code,
				"name": item.Name,
				"type": item.Type,
			})
		}
	}

	if len(filtered) == 0 {
		return helpers.Response(c, 404, "Not Found", "Tidak ada data ditemukan untuk tipe tersebut", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", filtered, nil)
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
