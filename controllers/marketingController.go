package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateMarketing - Create new marketing data (tanpa category dan status)
func CreateMarketing(c *fiber.Ctx) error {
	var body struct {
		Title       string `json:"title"`       // Judul Promosi
		StartDate   string `json:"start_date"`  // dd/mm/yyyy HH:mm
		EndDate     string `json:"end_date"`    // dd/mm/yyyy HH:mm  
		Broadcast   string `json:"broadcast"`   // Fifth broadcast
		Description string `json:"description"` // Deskripsi
		Image       string `json:"image"`       // Gambar (optional)
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Validate required fields
	if body.Title == "" || body.StartDate == "" || body.EndDate == "" || body.Description == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Title, start date, end date, and description are required", nil, nil)
	}

	// Set location to UTC untuk menghindari timezone shift
	loc, _ := time.LoadLocation("UTC")

	// Parse dates dengan format: "dd/mm/yyyy HH:mm" (24-hour format)
	startDate, err := time.ParseInLocation("02/01/2006 15:04", body.StartDate, loc)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid start date format (dd/mm/yyyy HH:mm)", nil, nil)
	}

	endDate, err := time.ParseInLocation("02/01/2006 15:04", body.EndDate, loc)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid end date format (dd/mm/yyyy HH:mm)", nil, nil)
	}

	// Validate dates
	if endDate.Before(startDate) {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "End date cannot be before start date", nil, nil)
	}

	marketing := models.Marketing{
		Title:       body.Title,
		StartDate:   startDate,
		EndDate:     endDate,
		Broadcast:   body.Broadcast,
		Description: body.Description,
		Image:       body.Image,
	}

	if err := configs.DB.Create(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to create marketing data", nil, nil)
	}

	return helpers.Response(c, 201, "Success", "Marketing data created successfully", marketing, nil)
}

// GetMarketingList - Get all marketing data with filtering (tanpa category dan status)
func GetMarketingList(c *fiber.Ctx) error {
	var query struct {
		StartDate  string `query:"start_date"`
		EndDate    string `query:"end_date"`
		Search     string `query:"search"`
		Page       int    `query:"page"`
		Limit      int    `query:"limit"`
	}

	if err := c.QueryParser(&query); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid query parameters", nil, nil)
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	var marketing []models.Marketing
	dbQuery := configs.DB

	// Apply date filter
	if query.StartDate != "" && query.EndDate != "" {
		startDate, err1 := time.Parse("02/01/2006", query.StartDate)
		endDate, err2 := time.Parse("02/01/2006", query.EndDate)
		
		if err1 == nil && err2 == nil {
			// Adjust end date to end of day
			endDate = endDate.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
			dbQuery = dbQuery.Where("start_date >= ? AND end_date <= ?", startDate, endDate)
		}
	}

	// Apply search filter (by title/description/broadcast)
	if query.Search != "" {
		search := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR description LIKE ? OR broadcast LIKE ?", search, search, search)
	}

	var total int64
	dbQuery.Model(&models.Marketing{}).Count(&total)

	// Get data dengan pagination dan order terbaru
	if err := dbQuery.Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to fetch marketing data", nil, nil)
	}

	// Format response
	var formattedMarketing []map[string]interface{}
	for i, item := range marketing {
		formattedMarketing = append(formattedMarketing, map[string]interface{}{
			"no":           i + 1 + offset,
			"id":           item.Id,
			"image":        item.Image,
			"title":        item.Title,
			"start_date":   helpers.FormatDateWithTime(item.StartDate),
			"end_date":     helpers.FormatDateWithTime(item.EndDate),
			"broadcast":    item.Broadcast,
			"description":  item.Description,
			"created_at":   item.CreatedAt,
			"updated_at":   item.UpdatedAt,
		})
	}

	response := map[string]interface{}{
		"marketing": formattedMarketing,
		"page":      query.Page,
		"limit":     query.Limit,
		"total":     total,
	}

	return helpers.Response(c, 200, "Success", "Marketing data retrieved successfully", response, nil)
}

// UpdateMarketing - Update marketing data (tanpa category dan status)
func UpdateMarketing(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid marketing ID", nil, nil)
	}

	var body struct {
		Title       string `json:"title"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		Broadcast   string `json:"broadcast"`
		Description string `json:"description"`
		Image       string `json:"image"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	var marketing models.Marketing
	if err := configs.DB.First(&marketing, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Marketing data not found", nil, nil)
	}

	// Set location to UTC
	loc, _ := time.LoadLocation("UTC")

	// Update fields
	if body.Title != "" {
		marketing.Title = body.Title
	}

	if body.StartDate != "" {
		startDate, err := time.ParseInLocation("02/01/2006 15:04", body.StartDate, loc)
		if err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid start date format (dd/mm/yyyy HH:mm)", nil, nil)
		}
		marketing.StartDate = startDate
	}

	if body.EndDate != "" {
		endDate, err := time.ParseInLocation("02/01/2006 15:04", body.EndDate, loc)
		if err != nil {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid end date format (dd/mm/yyyy HH:mm)", nil, nil)
		}
		marketing.EndDate = endDate
	}

	// Validate dates jika kedua date diupdate
	if body.StartDate != "" && body.EndDate != "" {
		if marketing.EndDate.Before(marketing.StartDate) {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "End date cannot be before start date", nil, nil)
		}
	}

	if body.Broadcast != "" {
		marketing.Broadcast = body.Broadcast
	}
	if body.Description != "" {
		marketing.Description = body.Description
	}
	if body.Image != "" {
		marketing.Image = body.Image
	}

	if err := configs.DB.Save(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update marketing data", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Marketing data updated successfully", marketing, nil)
}

// DeleteMarketing - Delete marketing data
func DeleteMarketing(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid marketing ID", nil, nil)
	}

	var marketing models.Marketing
	if err := configs.DB.First(&marketing, id).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "Marketing data not found", nil, nil)
	}

	if err := configs.DB.Delete(&marketing).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to delete marketing data", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Marketing data deleted successfully", nil, nil)
}