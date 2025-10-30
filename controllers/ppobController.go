package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	// "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	// "gorm.io/gorm"
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
	bicaraStr := c.Query("bicara") // hasilnya string, misal "true" atau "false"
	bicara, _ := strconv.ParseBool(bicaraStr)
	streamingStr := c.Query("streaming") // hasilnya string, misal "true" atau "false"
	streaming, _ := strconv.ParseBool(streamingStr)

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

		filtered := []models.ProductListPrepaid{}
		for _, item := range uniqueEtoll {
			key := strings.ToLower(item.ProductOperator)
			if allowed[key] {
				filtered = append(filtered, models.ProductListPrepaid{
					ProductDescription: item.ProductDescription,
					ProductOperator:    key,
					IconURL:            item.IconURL,
				})
			}
		}

		return helpers.Response(c, 200, "Success", "Data etoll berhasil diambil", filtered, nil)
	}

	if typ == "voucher" && operator == "" {
		uniqueEtoll := UniqueEtollByDescription(result.Data.Pricelist)

		filtered := []models.ProductListPrepaid{}
		for _, item := range uniqueEtoll {
			key := strings.ToLower(item.ProductOperator)
			filtered = append(filtered, models.ProductListPrepaid{
				ProductDescription: item.ProductDescription,
				ProductOperator:    key,
				IconURL:            item.IconURL,
			})
		}

		return helpers.Response(c, 200, "Success", "Data voucher berhasil diambil", filtered, nil)
	}

	// --- Inquiry Game ---
	if typ == "game" && operator == "" {
		var filtered []models.ProductListPrepaid

		if streaming {
			// --- Mode Streaming ---
			// Filter dulu dari Pricelist yang punya kategori "streaming"
			var streamingList []models.ProductPrepaid
			for _, item := range result.Data.Pricelist {
				if strings.EqualFold(strings.TrimSpace(item.ProductCategory), "streaming") {
					streamingList = append(streamingList, item)
				}
			}

			// Hilangkan duplikat
			uniqueStreaming := UniqueEtollByDescription(streamingList)

			for _, item := range uniqueStreaming {
				filtered = append(filtered, models.ProductListPrepaid{
					ProductDescription: item.ProductDescription,
					ProductOperator:    item.ProductOperator,
					IconURL:            item.IconURL,
				})
			}

			return helpers.Response(c, 200, "Success", "Data streaming berhasil diambil", filtered, nil)
		}

		// --- Mode Game Populer ---
		uniqueGames := UniqueEtollByDescription(result.Data.Pricelist)

		popularGames := map[string]bool{
			"mobile_legend":       true,
			"free_fire":           true,
			"pubg_mobile":         true,
			"pubg_mobile_global":  true,
			"genshin_impact":      true,
			"honkai_star_rail":    true,
			"valorant":            true,
			"call_of_duty_mobile": true,
			"clash_of_clans":      true,
			"arena_of_valor":      true,
			"roblox":              true,
			"point_blank":         true,
		}

		for _, item := range uniqueGames {
			if popularGames[strings.ToLower(item.ProductOperator)] {
				filtered = append(filtered, models.ProductListPrepaid{
					ProductDescription: item.ProductDescription,
					ProductOperator:    item.ProductOperator,
					IconURL:            item.IconURL,
				})
			}
		}

		return helpers.Response(c, 200, "Success", "Data game populer berhasil diambil", filtered, nil)
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
	if bicara && Number != "" {
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

		// 🟢 Filter hanya product_category == "bicara"
		filtered := []models.ProductPrepaid{}
		for _, item := range result.Data.Pricelist {
			if strings.ToLower(item.ProductCategory) == "bicara" &&
				(strings.Contains(strings.ToLower(item.ProductDescription), operatorName) ||
					strings.Contains(strings.ToLower(item.ProductCode), operatorName)) {
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

		if inquiryResult.Data.RC != "00" || inquiryResult.Data.Operator == "" {
			return helpers.Response(c, 400, "Failed", inquiryResult.Data.Message, nil, nil)
		}

		operatorName := strings.ToLower(inquiryResult.Data.Operator)

		// 🧩 Filter hanya produk yang cocok dengan operator, dan TIDAK "bicara"
		filtered := []models.ProductPrepaid{}
		for _, item := range result.Data.Pricelist {
			if (strings.Contains(strings.ToLower(item.ProductCategory), operatorName) ||
				strings.Contains(strings.ToLower(item.ProductDescription), operatorName)) &&
				strings.ToLower(item.ProductCategory) != "bicara" {
				filtered = append(filtered, item)
			}
		}

		if len(filtered) == 0 {
			return helpers.Response(c, 400, "Failed", fmt.Sprintf("Tidak ada produk pulsa untuk operator %s", operatorName), nil, nil)
		}

		combined := models.CombinedPrepaidResponse{
			Customer:  inquiryResult.Data,
			Pricelist: filtered,
		}

		return helpers.Response(c, 200, "Success", fmt.Sprintf("Data Pulsa %s berhasil digabungkan", strings.Title(operatorName)), combined, nil)
	}

	// --- Inquiry Data ---
	if typ == "data" && Number != "" {
		inquiryReq := map[string]string{
			"username":    username,
			"customer_id": Number,
			"sign":        MakeSignPricelist("op"),
		}
		inquiryBody, _ := json.Marshal(inquiryReq)
		inquiryResp, err := http.Post("https://prepaid.iak.dev/api/check-operator", "application/json", bytes.NewBuffer(inquiryBody))
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal request inquiry Data", nil, nil)
		}
		defer inquiryResp.Body.Close()

		inquiryData, _ := io.ReadAll(inquiryResp.Body)
		var inquiryResult models.InquiryPulsaResponse
		if err := json.Unmarshal(inquiryData, &inquiryResult); err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal decode inquiry Data", nil, nil)
		}

		// ❌ Jika nomor tidak valid
		if inquiryResult.Data.RC != "00" || inquiryResult.Data.Operator == "" {
			return helpers.Response(c, 400, "Failed", inquiryResult.Data.Message, nil, nil)
		}

		operatorName := strings.ToLower(inquiryResult.Data.Operator)

		// ✅ Daftar resmi operator internet
		operatorMapping := map[string]string{
			"axis":      "axis_paket_internet",
			"telkomsel": "telkomsel_paket_internet",
			"indosat":   "indosat_paket_internet",
			"smartfren": "smartfren_paket_internet",
			"tri":       "tri_paket_internet",
			"3":         "tri_paket_internet", // alias tri
			"xl":        "xl_paket_internet",
		}

		// ✅ Tentukan kategori target berdasarkan hasil inquiry
		targetCategory, ok := operatorMapping[operatorName]
		if !ok {
			return helpers.Response(c, 400, "Failed", fmt.Sprintf("Operator %s tidak dikenali", operatorName), nil, nil)
		}

		// 🔁 Update URL endpoint agar diarahkan ke kategori operator mapping
		url = fmt.Sprintf("https://prepaid.iak.dev/api/pricelist/%s/%s", typ, targetCategory)

		// 🔁 Request ulang daftar produk sesuai kategori mapping
		resp2, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal request API external", nil, nil)
		}
		defer resp2.Body.Close()

		body2, _ := io.ReadAll(resp2.Body)
		var result2 models.PrepaidResponse
		if err := json.Unmarshal(body2, &result2); err != nil {
			return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
		}

		if result2.Data.RC != "00" {
			return helpers.Response(c, 400, "Failed", result2.Data.Message, nil, nil)
		}

		// ✅ Filter hanya produk aktif dari hasil mapping
		filtered := []models.ProductPrepaid{}
		for _, item := range result2.Data.Pricelist {
			if strings.ToLower(item.Status) == "active" {
				filtered = append(filtered, item)
			}
		}

		if len(filtered) == 0 {
			return helpers.Response(c, 400, "Failed", fmt.Sprintf("Tidak ada produk untuk operator %s", operatorName), nil, nil)
		}

		// ✅ Gabungkan hasil inquiry + produk
		combined := models.CombinedPrepaidResponse{
			Customer:  inquiryResult.Data,
			Pricelist: filtered,
		}

		return helpers.Response(c, 200, "Success", fmt.Sprintf("Data %s berhasil digabungkan dari /data/%s", strings.Title(operatorName), targetCategory), combined, nil)
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", result.Data.Pricelist, nil)
}

func UniqueEtollByDescription(products []models.ProductPrepaid) []models.ProductListPrepaid {
	seen := make(map[string]bool)
	unique := make([]models.ProductListPrepaid, 0)

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

			unique = append(unique, models.ProductListPrepaid{
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

// topup prepaid and save to history
func TopupPrepaid(c *fiber.Ctx) error {
	username := os.Getenv("IDENTITY")

	var reqBody struct {
		RefID         string `json:"ref_id"`
		UserID        uint   `json:"user_id"`
		ProductCode   string `json:"product_code"`
		ProductName   string `json:"product_name"`
		ProductPrice  string  `json:"product_price"`
		ProductType   string `json:"product_type"`
		UserNumber    string `json:"user_number"`
		TotalPrice    string  `json:"total_price"`
		StroomToken   string `json:"stroom_token"`
		BillingPeriod string `json:"billing_period"`
		Year          string `json:"year"`
		Province      string `json:"province"`
		Region        string `json:"region"`
	}
	if err := c.BodyParser(&reqBody); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal membaca body", nil, nil)
	}

	// Ambil margin PPOB
	// var settings models.Ppob
	// if err := configs.DB.First(&settings).Error; err != nil {
	// 	return helpers.Response(c, 400, "Failed", "Gagal mengambil margin", nil, nil)
	// }
	// margin := float64(settings.Margin)

	// Request ke API eksternal
	requestBody := models.ExternalRequestTopup{
		Username:    username,
		Sign:        MakeSignPricelist(reqBody.RefID),
		RefId:       reqBody.RefID,
		CustomerId:  reqBody.UserNumber,
		ProductCode: reqBody.ProductCode,
	}
	jsonBody, _ := json.Marshal(requestBody)

	resp, err := http.Post("https://prepaid.iak.dev/api/top-up", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal request API eksternal", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result models.PrepaidResponseTopup
	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	// // Update saldo perusahaan
	// marginAmount := helpers.RoundToNearest(float64(reqBody.ProductPrice) * (margin / 100))
	// var company models.Company
	// if err := configs.DB.First(&company).Error; err != nil {
	// 	if errors.Is(err, gorm.ErrRecordNotFound) {
	// 		company = models.Company{Balance: 0}
	// 		configs.DB.Create(&company)
	// 	} else {
	// 		return helpers.Response(c, 400, "Failed", "Gagal mengambil data company", nil, nil)
	// 	}
	// }

	// company.Balance += int(marginAmount)
	// if err := configs.DB.Save(&company).Error; err != nil {
	// 	return helpers.Response(c, 400, "Failed", "Gagal update saldo perusahaan", nil, nil)
	// }

	// Simpan riwayat ke database
	history := models.HistoryModel{
		UserID:        reqBody.UserID,
		RefID:         reqBody.RefID,
		ProductName:   reqBody.ProductName,
		ProductPrice:  reqBody.ProductPrice,
		ProductType:   reqBody.ProductType,
		UserNumber:    reqBody.UserNumber,
		TotalPrice:    reqBody.TotalPrice,
		StroomToken:   reqBody.StroomToken,
		BillingPeriod: reqBody.BillingPeriod,
		Year:          reqBody.Year,
		Province:      reqBody.Province,
		Region:        reqBody.Region,
		Status:        "Proses",
	}
	if err := configs.DB.Create(&history).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Gagal menyimpan riwayat transaksi", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Transaksi berhasil dan riwayat tersimpan", result.Data, nil)
}
