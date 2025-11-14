package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)
// CREATE
func CreateChildBank(c *fiber.Ctx) error {
	// bikin struct body langsung di dalam function
	var body struct {
		Subdistrict  string  `json:"subdistrict"`
		RT           string  `json:"rt"`
		RW           string  `json:"rw"`
		Address      string  `json:"address"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		ParentBankID uint    `json:"parentBank_id"`
		Norek        uint    `json:"norek"`
	}

	// Parse body JSON
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	// Validate latitude and longitude
	if body.Latitude < -90 || body.Latitude > 90 {
		return helpers.Response(c, 400, "Failed", "Invalid latitude value (must be between -90 and 90)", nil, nil)
	}
	if body.Longitude < -180 || body.Longitude > 180 {
		return helpers.Response(c, 400, "Failed", "Invalid longitude value (must be between -180 and 180)", nil, nil)
	}

	// mapping ke model ChildBank
	childBank := models.ChildBank{
		Subdistrict:  body.Subdistrict,
		RT:           body.RT,
		RW:           body.RW,
		Address:      body.Address,
		Latitude:     body.Latitude,
		Longitude:    body.Longitude,
		ParentBankID: body.ParentBankID,
		Norek:        body.Norek,
	}

	// simpan ke database
	if err := configs.DB.Create(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	res := models.ChildBank{
		Id:           childBank.Id,
		Subdistrict:  childBank.Subdistrict,
		RT:           childBank.RT,
		RW:           childBank.RW,
		Address:      childBank.Address,
		Latitude:     childBank.Latitude,
		Longitude:    childBank.Longitude,
		ParentBankID: childBank.ParentBankID,
		Norek:        childBank.Norek,
	}

	return helpers.Response(c, 201, "Success", "Child Bank created successfully", res, nil)
}

// READ ALL dengan Pagination
func GetAllChildBanks(c *fiber.Ctx) error {
	var query struct {
		Search      string `query:"search"`
		Page        int    `query:"page"`
		Limit       int    `query:"limit"`
		Subdistrict string `query:"subdistrict"`
		ParentBankID string `query:"parent_bank_id"`
	}

	if err := c.QueryParser(&query); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid query parameters", nil, nil)
	}

	// Set default values
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	var childBanks []models.ChildBank
	dbQuery := configs.DB.Preload("ParentBank")

	// Apply search filter
	if query.Search != "" {
		search := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("subdistrict LIKE ? OR address LIKE ? OR rt LIKE ? OR rw LIKE ?", 
			search, search, search, search)
	}

	// Filter by subdistrict
	if query.Subdistrict != "" {
		dbQuery = dbQuery.Where("subdistrict = ?", query.Subdistrict)
	}

	// Filter by parent_bank_id
	if query.ParentBankID != "" {
		dbQuery = dbQuery.Where("parent_bank_id = ?", query.ParentBankID)
	}

	// Count total records
	var total int64
	dbQuery.Model(&models.ChildBank{}).Count(&total)

	// Calculate total pages
	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	// Get data dengan pagination dan order
	if err := dbQuery.Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&childBanks).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch child banks data", nil, nil)
	}

	// Format response dengan nomor urut
	var formattedChildBanks []map[string]interface{}
	for i, bank := range childBanks {
		formattedBank := map[string]interface{}{
			"no":           i + 1 + offset,
			"id":           bank.Id,
			"subdistrict":  bank.Subdistrict,
			"rt":           bank.RT,
			"rw":           bank.RW,
			"address":      bank.Address,
			"latitude":     bank.Latitude,
			"longitude":    bank.Longitude,
			"norek":        bank.Norek,
			"parent_bank":  bank.ParentBank,
			"created_at":   bank.CreatedAt,
			"updated_at":   bank.UpdatedAt,
		}
		formattedChildBanks = append(formattedChildBanks, formattedBank)
	}

	// Format meta data sederhana
	meta := map[string]interface{}{
		"limit": query.Limit,
		"page":  query.Page,
		"pages": totalPages,
		"total": total,
	}

	response := map[string]interface{}{
		"child_banks": formattedChildBanks,
		"meta":        meta,
	}

	return helpers.Response(c, fiber.StatusOK, "Success", "Child banks retrieved successfully", response, nil)
}

// READ BY ID
func GetChildBankById(c *fiber.Ctx) error {
	id := c.Params("id")
	var childBank models.ChildBank

	if err := configs.DB.Preload("ParentBank").First(&childBank, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Child Bank retrieved successfully", childBank, nil)
}
// Function for Admin in Web (Update Child Bank)
func UpdateChildBank(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Subdistrict  string  `json:"subdistrict"`
		RT           string  `json:"rt"`
		RW           string  `json:"rw"`
		Address      string  `json:"address"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		ParentBankID uint    `json:"parentBank_id"`
		Norek        uint    `json:"norek"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to read body", nil, nil)
	}

	// Validate latitude and longitude
	if body.Latitude < -90 || body.Latitude > 90 {
		return helpers.Response(c, 400, "Failed", "Invalid latitude value (must be between -90 and 90)", nil, nil)
	}
	if body.Longitude < -180 || body.Longitude > 180 {
		return helpers.Response(c, 400, "Failed", "Invalid longitude value (must be between -180 and 180)", nil, nil)
	}

	var childBank models.ChildBank
	if err := configs.DB.First(&childBank, id).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
	}

	childBank.Subdistrict = body.Subdistrict
	childBank.RT = body.RT
	childBank.RW = body.RW
	childBank.Address = body.Address
	childBank.Latitude = body.Latitude
	childBank.Longitude = body.Longitude
	childBank.ParentBankID = body.ParentBankID
	childBank.Norek = body.Norek
	childBank.UpdatedAt = time.Now()

	if err := configs.DB.Save(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to update child bank", nil, nil)
	}

	res := models.ChildBank{
		Id:           childBank.Id,
		Subdistrict:  childBank.Subdistrict,
		RT:           childBank.RT,
		RW:           childBank.RW,
		Address:      childBank.Address,
		Latitude:     childBank.Latitude,
		Longitude:    childBank.Longitude,
		ParentBankID: childBank.ParentBankID,
		Norek:        childBank.Norek,
	}

	return helpers.Response(c, 200, "Success", "Child Bank updated successfully", res, nil)
}

// DELETE
func DeleteChildBank(c *fiber.Ctx) error {
	id := c.Params("id")
	var childBank models.ChildBank

	if err := configs.DB.First(&childBank, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Child Bank not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	if err := configs.DB.Delete(&childBank).Error; err != nil {
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Child Bank deleted successfully", nil, nil)	
}

func GetUserChildBankByIDC(c *fiber.Ctx) error {
	// get email and pass from req body
	id := c.Params("id")

	var user models.User
	// TAMBAHKAN Preload untuk ParentBank dan ChildBank
	configs.DB.
		Preload("Division").
		Preload("Role").
		Preload("Plan").
		Preload("ParentBank"). // Tambahkan ini
		Preload("ChildBank").  // Tambahkan ini
		First(&user, id)

	if user.Id == 0 {
		return helpers.Response(c, 400, "Failed", "User not found", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Data User found", user, nil)
}
