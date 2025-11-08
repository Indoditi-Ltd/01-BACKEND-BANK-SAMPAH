package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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

	sign := helpers.MakeSignPricelist("pl")

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
			"sign":        helpers.MakeSignPricelist(NumberPLN),
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
			"sign":        helpers.MakeSignPricelist(NumberOVO),
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
			"sign":        helpers.MakeSignPricelist("op"),
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
			"sign":        helpers.MakeSignPricelist("op"),
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
			"sign":        helpers.MakeSignPricelist("op"),
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

// topup prepaid and save to history
func TopupPrepaid(c *fiber.Ctx) error {
	username := os.Getenv("IDENTITY")

	var reqBody struct {
		RefID         string `json:"ref_id"`
		UserID        uint   `json:"user_id"`
		ProductCode   string `json:"product_code"`
		ProductName   string `json:"product_name"`
		ProductPrice  string `json:"product_price"`
		ProductType   string `json:"product_type"`
		UserNumber    string `json:"user_number"`
		TotalPrice    string `json:"total_price"`
		StroomToken   string `json:"stroom_token"`
		BillingPeriod string `json:"billing_period"`
		Year          string `json:"year"`
		Province      string `json:"province"`
		Region        string `json:"region"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return helpers.Response(c, 400, "Failed", "Gagal membaca body", nil, nil)
	}

	// ⚡ PERBAIKAN: Gunakan TotalPrice yang sudah dalam format angka saja
	// TotalPrice: "11500" (tanpa "Rp.")
	productPrice, err := strconv.Atoi(reqBody.TotalPrice)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Format total price tidak valid: "+reqBody.TotalPrice, nil, nil)
	}

	// Start database transaction
	tx := configs.DB.Begin()

	// 1. Cek dan potong saldo user di awal (dengan lock untuk avoid race condition)
	var user models.User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, reqBody.UserID).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 404, "Failed", "User tidak ditemukan", nil, nil)
	}

	// Validasi saldo user cukup
	if user.Balance < productPrice {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed",
			fmt.Sprintf("Saldo tidak cukup. Saldo anda: Rp. %d, Dibutuhkan: Rp. %d",
				user.Balance, productPrice), nil, nil)
	}

	// Potong saldo user di awal
	user.Balance -= productPrice
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal memotong saldo user", nil, nil)
	}

	// 2. Request ke API eksternal
	requestBody := models.ExternalRequestTopup{
		Username:    username,
		Sign:        helpers.MakeSignPricelist(reqBody.RefID),
		RefId:       reqBody.RefID,
		CustomerId:  reqBody.UserNumber,
		ProductCode: reqBody.ProductCode,
	}
	jsonBody, _ := json.Marshal(requestBody)

	resp, err := http.Post("https://prepaid.iak.dev/api/top-up", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		// ❌ Jika API request gagal, kembalikan saldo user
		user.Balance += productPrice
		tx.Save(&user)
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Gagal request API eksternal", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result models.PrepaidResponseTopup
	if err := json.Unmarshal(body, &result); err != nil {
		// ❌ Jika decode response gagal, kembalikan saldo user
		user.Balance += productPrice
		tx.Save(&user)
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Gagal decode response API", nil, nil)
	}

	// ⚠️ Jika response dari API gagal (misalnya status = 2 atau pesan MAXIMUM ...),
	// maka kembalikan saldo user
	if result.Data.Status == 2 || strings.Contains(strings.ToUpper(result.Data.Message), "MAXIMUM 1 NUMBER 1 TIME IN 1 DAY") {
		user.Balance += productPrice
		tx.Save(&user)
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", result.Data.Message, result.Data, nil)
	}

	// 3. Simpan riwayat ke database dengan status PROSES
	history := models.HistoryModel{
		UserID:        reqBody.UserID,
		RefID:         reqBody.RefID,
		ProductName:   reqBody.ProductName,
		ProductPrice:  reqBody.ProductPrice, // Tetap simpan yang asli dengan "Rp." untuk display
		ProductType:   reqBody.ProductType,
		UserNumber:    reqBody.UserNumber,
		TotalPrice:    reqBody.TotalPrice, // Simpan yang sudah angka saja "11500"
		StroomToken:   reqBody.StroomToken,
		BillingPeriod: reqBody.BillingPeriod,
		Year:          reqBody.Year,
		Province:      reqBody.Province,
		Region:        reqBody.Region,
		Status:        "PROSES", // Menunggu callback
	}
	if err := tx.Create(&history).Error; err != nil {
		// ❌ Jika gagal simpan history, kembalikan saldo user
		user.Balance += productPrice
		tx.Save(&user)
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal menyimpan riwayat transaksi", nil, nil)
	}

	// Commit transaction
	tx.Commit()

	// Log untuk debugging
	fmt.Printf("✅ TopupPrepaid berhasil - UserID: %d, Amount: Rp. %d, Saldo tersisa: Rp. %d, Status: PROSES\n",
		reqBody.UserID, productPrice, user.Balance)

	return helpers.Response(c, 200, "Success", "Transaksi diproses, menunggu konfirmasi", result.Data, nil)
}

func CallbackPrepaid(c *fiber.Ctx) error {
	// Struktur sesuai dengan JSON callback dari API
	var body struct {
		Data struct {
			RefID       string `json:"ref_id"`
			Status      string `json:"status"`
			ProductCode string `json:"product_code"`
			CustomerID  string `json:"customer_id"`
			Price       string `json:"price"`
			Message     string `json:"message"`
			SN          string `json:"sn"`
			PIN         string `json:"pin"`
			Balance     string `json:"balance"`
			TrID        string `json:"tr_id"`
			RC          string `json:"rc"`
			Sign        string `json:"sign"`
		} `json:"data"`
	}

	// Parse JSON body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "invalid request body",
			"error":   err.Error(),
		})
	}

	data := body.Data

	// Validasi ref_id
	if strings.TrimSpace(data.RefID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "missing ref_id in callback data",
		})
	}

	// Start database transaction
	tx := configs.DB.Begin()

	// 1. Cari history transaksi berdasarkan ref_id
	var history models.HistoryModel
	if err := tx.Where("ref_id = ?", data.RefID).First(&history).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "transaction not found",
		})
	}

	// 2. Siapkan data yang akan diupdate
	updateData := map[string]any{
		"status": data.Message, // "SUCCESS" atau "FAILED"
	}

	// ⚡ PLN memiliki stroom_token, non-PLN kosong
	if strings.Contains(strings.ToLower(data.ProductCode), "pln") {
		updateData["stroom_token"] = data.SN
	} else {
		updateData["stroom_token"] = ""
	}

	// 3. Jika status SUCCESS (status = "1" dan rc = "00")
	if data.Status == "1" && data.RC == "00" {
		// ⚡ PERBAIKAN: Gunakan TotalPrice yang sudah dalam format angka
		userPaidPrice, err := strconv.Atoi(history.TotalPrice)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid total price format in history: " + history.TotalPrice,
			})
		}

		// Convert real price dari callback ke int
		realPrice, err := strconv.Atoi(data.Price)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid price format in callback: " + data.Price,
			})
		}

		// 3a. Hitung margin (selisih antara yang dibayar user vs real price dari callback)
		marginAmount := userPaidPrice - realPrice

		if marginAmount > 0 {
			// 3b. Tambah margin ke company balance
			var company models.Company
			if err := tx.First(&company).Error; err != nil {
				// Jika company tidak ada, buat baru
				company = models.Company{Balance: marginAmount}
				if err := tx.Create(&company).Error; err != nil {
					tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"status":  "error",
						"message": "failed to create company record",
					})
				}
			} else {
				// Jika company sudah ada, tambah balance
				company.Balance += marginAmount
				if err := tx.Save(&company).Error; err != nil {
					tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"status":  "error",
						"message": "failed to update company balance",
					})
				}
			}

			// Hitung persentase margin
			marginPercentage := (float64(marginAmount) / float64(realPrice)) * 100

			fmt.Printf("💰 SUCCESS - Margin calculation:\n")
			fmt.Printf("   User paid: Rp. %d\n", userPaidPrice)
			fmt.Printf("   IAK price: Rp. %d\n", realPrice)
			fmt.Printf("   Margin: Rp. %d (%.1f%%)\n", marginAmount, marginPercentage)
			fmt.Printf("   Added to company balance\n")
		} else if marginAmount == 0 {
			fmt.Printf("ℹ️ SUCCESS - No margin. User paid: Rp. %d, IAK price: Rp. %d\n",
				userPaidPrice, realPrice)
		} else {
			fmt.Printf("⚠️ SUCCESS - Negative margin detected! User paid less than IAK price\n")
			fmt.Printf("   User paid: Rp. %d, IAK price: Rp. %d\n", userPaidPrice, realPrice)
		}

		// Update history status
		if err := tx.Model(&history).Updates(updateData).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "failed to update history",
				"error":   err.Error(),
			})
		}

		fmt.Printf("✅ Transaction SUCCESS - RefID: %s\n", data.RefID)

	} else {
		// 4. Jika status FAILED, kembalikan saldo ke user
		fmt.Printf("❌ Transaction FAILED - Status: %s, RC: %s, Message: %s\n",
			data.Status, data.RC, data.Message)

		// ⚡ PERBAIKAN: Gunakan TotalPrice untuk refund
		productPrice, err := strconv.Atoi(history.TotalPrice)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "invalid total price format for refund: " + history.TotalPrice,
			})
		}

		// Kembalikan saldo ke user
		var user models.User
		if err := tx.First(&user, history.UserID).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "user not found for refund",
			})
		}

		user.Balance += productPrice
		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "failed to refund user balance",
			})
		}

		// Update history status ke FAILED
		if err := tx.Model(&history).Updates(updateData).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "failed to update history status",
				"error":   err.Error(),
			})
		}

		fmt.Printf("💰 Refund processed - UserID: %d, Amount: Rp. %d, New Balance: Rp. %d\n",
			user.Id, productPrice, user.Balance)
	}

	// 5. Commit transaction
	tx.Commit()

	// Log untuk debugging
	fmt.Println("✅ CallbackPrepaid processed at:", time.Now().Format("02-01-2006 15:04:05"))
	fmt.Printf("📦 Callback data: RefID: %s, Status: %s, RC: %s, Message: %s\n",
		data.RefID, data.Status, data.RC, data.Message)

	// Kirim respon sukses
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "callback processed successfully",
		"data": fiber.Map{
			"ref_id":       data.RefID,
			"status":       data.Status,
			"rc":           data.RC,
			"message":      data.Message,
			"processed_at": time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

func GetHistoryByRefID(c *fiber.Ctx) error {
	// Ambil ref_id dari query parameter
	refID := c.Query("ref_id")

	if refID == "" {
		return helpers.Response(c, 400, "Failed", "ref_id is required", nil, nil)
	}

	var history models.HistoryModel

	// Cari data berdasarkan ref_id
	if err := configs.DB.
		Preload("User").
		Where("ref_id = ?", refID).
		First(&history).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helpers.Response(c, 404, "Failed", "History not found", nil, nil)
		}

		return helpers.Response(c, 500, "Failed", "Failed to fetch history", nil, nil)
	}

	// Format tanggal menjadi dd-mm-YYYY HH:MM
	formattedHistory := fiber.Map{
		"id":             history.Id,
		"created_at":     history.CreatedAt.Format("02-01-2006 15:04"),
		"updated_at":     history.UpdatedAt.Format("02-01-2006 15:04"),
		"deleted_at":     history.DeletedAt,
		"user":           history.User,
		"ref_id":         history.RefID,
		"product_name":   history.ProductName,
		"product_price":  history.ProductPrice,
		"product_type":   history.ProductType,
		"user_number":    history.UserNumber,
		"total_price":    history.TotalPrice,
		"stroom_token":   history.StroomToken,
		"billing_period": history.BillingPeriod,
		"year":           history.Year,
		"province":       history.Province,
		"region":         history.Region,
		"status":         history.Status,
	}

	// ✅ Respons sukses pakai helper
	return helpers.Response(c, 200, "Success", "History retrieved successfully", formattedHistory, nil)
}
