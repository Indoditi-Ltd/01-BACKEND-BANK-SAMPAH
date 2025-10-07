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

	"github.com/gofiber/fiber/v2"
)

func MakeSign() string {
	toSign := os.Getenv("IDENTITY") + os.Getenv("APIKEY") + "pl"
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
		// 🔹 jika sudah ada → update
		existing.Margin = body.Margin
		if err := configs.DB.Save(&existing).Error; err != nil {
			return helpers.Response(c, 400, "Failed", "Failed to update margin", nil, nil)
		}
		return helpers.Response(c, 200, "Success", "Margin updated successfully", existing, nil)
	}

	// 🔹 kalau belum ada → buat baru
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
	sign := MakeSign()

	fmt.Println("SIGN:", sign)

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

	url := "https://prepaid.iak.dev/api/pricelist"
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

	for i := range result.Data.Pricelist {
		price := result.Data.Pricelist[i].ProductPrice
		result.Data.Pricelist[i].ProductPrice = helpers.RoundToNearest(price * (1 + margin/100))
	}

	return helpers.Response(c, 200, "Success", "Data retrieved successfully", result.Data.Pricelist, nil)
}

// get list postpaid
func GetListPostpaid(c *fiber.Ctx) error {
	typ := c.Params("type")
	province := c.Query("province")
	username := os.Getenv("IDENTITY")
	sign := MakeSign()

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
