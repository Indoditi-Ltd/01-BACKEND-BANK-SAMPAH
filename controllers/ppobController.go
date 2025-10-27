package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func MakeSignPricelist(UniqueCode string) string {
	toSign := os.Getenv("IDENTITY") + os.Getenv("APIKEY") + UniqueCode
	h := md5.New()
	h.Write([]byte(toSign))
	return hex.EncodeToString(h.Sum(nil))
}

func CreateMargin(c *fiber.Ctx) error {
	var body struct {
		Margin int `json:"margin"`
	}

	// parsing body JSON
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to read body", nil, nil)
	}

	// cek apakah sudah ada data margin di tabel
	var existing models.Ppob
	if err := configs.DB.First(&existing).Error; err == nil {
		// jika sudah ada → update
		existing.Margin = body.Margin
		if err := configs.DB.Save(&existing).Error; err != nil {
			return helpers.Response(c, 400, "Failed", "Failed to update margin", nil, nil)
		}
		return helpers.Response(c, 200, "Success", "Margin updated successfully", existing, nil)
	}

	// kalau belum ada → buat baru
	ppob := models.Ppob{
		Margin: body.Margin,
	}
	if err := configs.DB.Create(&ppob).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to create margin", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Margin created successfully", ppob, nil)
}

// get list prepaid PPOB
func GetListPrepaid(c *fiber.Ctx) error {
	username := os.Getenv("IDENTITY")
	typ := c.Params("type")
	operator := c.Query("operator")
	NumberPLN := c.Query("numberPLN")
	NumberOVO := c.Query("numberOVO")
	Number := c.Query("numberPulsa")

	sign := MakeSignPricelist("pl")

	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}
	var settings models.Ppob
	if err := configs.DB.First(&settings).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal mengambil margin dari database", nil, nil)
	}
	margin := float64(settings.Margin)

	reqBody := models.ExternalRequestPrepaid{
		Status:   "all",
		Username: username,
		Sign:     sign,
	}
	jsonBody, _ := json.Marshal(reqBody)

	// url := "https://prepaid.iak.dev/api/pricelist/%s"
	url := fmt.Sprintf("https://prepaid.iak.dev/api/pricelist/%s/%s", typ, operator)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Failed request API external", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result models.PrepaidResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	if result.Data.RC != "00" {
		return helpers.Response(c, 400, "Failed", result.Data.Message, nil, nil)
	}

	if typ == "etoll" && operator == "" {
		for i := range result.Data.Pricelist {
			price := result.Data.Pricelist[i].ProductPrice
			result.Data.Pricelist[i].ProductPrice = helpers.RoundToNearest(price * (1 + margin/100))
		}

		uniqueEtoll := UniqueEtollByDescription(result.Data.Pricelist)

		allowed := map[string]bool{
			"alipay":                 false,
			"dana":                   true,
			"gopay_e-money":          true,
			"ovo":                    true,
			"shopee_pay":             true,
			"indomaret_card_e-money": true,
			"mandiri_e-toll":         true,
			"linkaja":                true,
		}

		filtered := []models.ProductEtollPrepaid{}
		for _, item := range uniqueEtoll {
			key := strings.ToLower(item.ProductOperator)
			if allowed[key] {
				filtered = append(filtered, models.ProductEtollPrepaid{
					ProductDescription: item.ProductDescription,
					ProductOperator:    key,
					IconURL:            item.IconURL,
				})
			}
		}

		return helpers.Response(c, 200, "Success", "Data etoll berhasil diambil", filtered, nil)
	}

	for i := range result.Data.Pricelist {
		price := result.Data.Pricelist[i].ProductPrice
		result.Data.Pricelist[i].ProductPrice = helpers.RoundToNearest(price * (1 + margin/100))
	}

	if typ == "pln" && NumberPLN != "" {
		inquiryReq := map[string]string{
			"username":    username,
			"customer_id": NumberPLN,
			"sign":        MakeSignPricelist(NumberPLN),
		}
		inquiryBody, _ := json.Marshal(inquiryReq)
		urlInqueryPLN := "https://prepaid.iak.dev/api/inquiry-pln"
		inquiryResp, err := http.Post(urlInqueryPLN, "application/json", bytes.NewBuffer(inquiryBody))

		if err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal request inquiry PLN", nil, nil)
		}
		defer inquiryResp.Body.Close()

		inquiryData, _ := io.ReadAll(inquiryResp.Body)
		var inquiryResult models.InquiryPLNResponse
		if err := json.Unmarshal(inquiryData, &inquiryResult); err != nil {
			return helpers.Response(c, 400, "Failed", "Destination number not found", nil, nil)
		}

		combined := models.CombinedPrepaidResponse{
			Customer:  inquiryResult.Data,
			Pricelist: result.Data.Pricelist,
		}
		return helpers.Response(c, 200, "Success", "Data PLN berhasil digabungkan", combined, nil)

	}

	// --- Jika OVO, lakukan Inquiry OVO ---
	if typ == "etoll" && operator == "ovo" && NumberOVO != "" {
		inquiryReq := map[string]string{
			"username":    username,
			"customer_id": NumberOVO,
			"sign":        MakeSignPricelist(NumberOVO),
		}
		inquiryBody, _ := json.Marshal(inquiryReq)
		inquiryResp, err := http.Post("https://prepaid.iak.dev/api/inquiry-ovo", "application/json", bytes.NewBuffer(inquiryBody))
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal request inquiry OVO", nil, nil)
		}
		defer inquiryResp.Body.Close()

		inquiryData, _ := io.ReadAll(inquiryResp.Body)
		var inquiryResult models.InquiryOVOResponse
		if err := json.Unmarshal(inquiryData, &inquiryResult); err != nil {
			return helpers.Response(c, 400, "Failed", "Destination number not found", nil, nil)
		}

		combined := models.CombinedPrepaidResponse{
			Customer:  inquiryResult.Data,
			Pricelist: result.Data.Pricelist,
		}
		return helpers.Response(c, 200, "Success", "Data OVO berhasil digabungkan", combined, nil)
	}

	// --- Inquiry Pulsa ---
	if typ == "pulsa" && Number != "" {
		inquiryReq := map[string]string{
			"username":    username,
			"customer_id": Number,
			"sign":        MakeSignPricelist("op"),
		}
		inquiryBody, _ := json.Marshal(inquiryReq)
		inquiryResp, err := http.Post("https://prepaid.iak.dev/api/check-operator", "application/json", bytes.NewBuffer(inquiryBody))
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal request inquiry Pulsa", nil, nil)
		}
		defer inquiryResp.Body.Close()

		inquiryData, _ := io.ReadAll(inquiryResp.Body)
		var inquiryResult models.InquiryPulsaResponse
		if err := json.Unmarshal(inquiryData, &inquiryResult); err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal decode inquiry Pulsa", nil, nil)
		}

		// 🧩 Jika inquiry gagal (rc bukan "00" atau operator kosong)
		if inquiryResult.Data.RC != "00" || inquiryResult.Data.Operator == "" {
			return helpers.Response(c, 400, "Failed", inquiryResult.Data.Message, nil, nil)
		}

		// Operator dari hasil inquiry
		operatorName := strings.ToLower(inquiryResult.Data.Operator)

		// Filter hanya produk dengan kategori/operator yang sama
		filtered := []models.ProductPrepaid{}
		for _, item := range result.Data.Pricelist {
			if strings.Contains(strings.ToLower(item.ProductCategory), operatorName) ||
				strings.Contains(strings.ToLower(item.ProductDescription), operatorName) {
				filtered = append(filtered, item)
			}
		}

		// Jika tidak ada produk yang cocok, beri pesan error
		if len(filtered) == 0 {
			return helpers.Response(c, 400, "Failed", fmt.Sprintf("Tidak ada produk untuk operator %s", operatorName), nil, nil)
		}

		// Gabungkan data operator + produk
		combined := models.CombinedPrepaidResponse{
			Customer:  inquiryResult.Data,
			Pricelist: filtered,
		}

		return helpers.Response(c, 200, "Success", fmt.Sprintf("Data Pulsa %s berhasil digabungkan", strings.Title(operatorName)), combined, nil)
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", result.Data.Pricelist, nil)
}

func UniqueEtollByDescription(products []models.ProductPrepaid) []models.ProductEtollPrepaid {
	seen := make(map[string]bool)
	unique := make([]models.ProductEtollPrepaid, 0)

	// Regex untuk hapus tanda kurung dan spasi, ganti dengan "_"
	re := regexp.MustCompile(`[()\s]+`)

	for _, p := range products {
		if !seen[p.ProductDescription] {
			seen[p.ProductDescription] = true

			// Bersihkan product description → jadi operator
			operator := re.ReplaceAllString(p.ProductDescription, "_")

			// Hapus underscore berlebih di awal/akhir dan tengah
			operator = strings.Trim(operator, "_")
			operator = strings.ReplaceAll(operator, "__", "_")

			// Ubah jadi huruf kecil semua
			operator = strings.ToLower(operator)

			unique = append(unique, models.ProductEtollPrepaid{
				ProductDescription: p.ProductDescription,
				ProductOperator:    operator,
				IconURL:            p.IconURL,
			})
		}
	}

	return unique
}

// get list postpaid
func GetListPostpaid(c *fiber.Ctx) error {
	typ := c.Params("type")
	province := c.Query("province")
	username := os.Getenv("IDENTITY")
	sign := MakeSignPricelist("pl")

	if username == "" || sign == "" {
		return helpers.Response(c, 400, "Failed", "Username or sign is Empty", nil, nil)
	}
	var settings models.Ppob
	if err := configs.DB.First(&settings).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal mengambil margin dari database", nil, nil)
	}
	margin := float64(settings.Margin)

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

	var url string
	if typ == "" {
		url = "https://testpostpaid.mobilepulsa.net/api/v1/bill/check/"
	} else {
		url = fmt.Sprintf("https://testpostpaid.mobilepulsa.net/api/v1/bill/check/%s", typ)
	}

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

	for i := range result.Data.Pasca {
		price := result.Data.Pasca[i].Fee
		result.Data.Pasca[i].Fee = helpers.RoundToNearest(price * (1 + margin/100))
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", result.Data.Pasca, nil)
}

func TopupPrepaid() {

}
