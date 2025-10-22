package controllers

import (
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func RegionProvince(c *fiber.Ctx) error {
	url := "https://ibnux.github.io/data-indonesia/provinsi.json"
	resp, err := http.Get(url)

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Cannot get data province", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result []models.RegionResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to decode API response", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "List of provinces", result, nil)
}

func RegionDistrict(c *fiber.Ctx) error {
	idPovince := c.Query("idProvince")
	url := fmt.Sprintf("https://ibnux.github.io/data-indonesia/kabupaten/%s.json", idPovince)
	resp, err := http.Get(url)

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Cannot get data province", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result []models.RegionResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to decode API response", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "List of provinces", result, nil)
}

func RegionSubDistrict(c *fiber.Ctx) error {
	idDistrict := c.Query("idDistrict")
	url := fmt.Sprintf("https://ibnux.github.io/data-indonesia/kecamatan/%s.json", idDistrict)
	resp, err := http.Get(url)

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Cannot get data province", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result []models.RegionResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to decode API response", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "List of provinces", result, nil)
}

func RegionVillage(c *fiber.Ctx) error {
	idSubdistrict := c.Query("idSubdistrict")
	url := fmt.Sprintf("https://ibnux.github.io/data-indonesia/kelurahan/%s.json", idSubdistrict)
	resp, err := http.Get(url)

	if err != nil {
		return helpers.Response(c, 400, "Failed", "Cannot get data province", nil, nil)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result []models.RegionResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to decode API response", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "List of provinces", result, nil)
}
