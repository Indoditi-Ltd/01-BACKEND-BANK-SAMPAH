package pickuprequest

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CheckNearbyBanks - Cek semua bank pembantu dalam radius 1KM (bukan hanya yang terdekat)
func CheckNearbyBanks(c *fiber.Ctx) error {
	var body struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Radius    float64 `json:"radius"` // Optional, default 1KM
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Failed to read body", nil, nil)
	}

	// Validasi coordinates
	if body.Latitude == 0 || body.Longitude == 0 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Latitude and longitude are required", nil, nil)
	}

	// Set default radius 1KM jika tidak diisi
	if body.Radius == 0 {
		body.Radius = 1.0
	}

	// Cari SEMUA bank pembantu dalam radius 1KM (bukan hanya 1)
	var nearbyChildBanks []models.ChildBank
	query := `
		SELECT *,
		(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * 
		cos(radians(longitude) - radians(?)) + 
		sin(radians(?)) * sin(radians(latitude)))) AS distance
		FROM child_banks 
		WHERE deleted_at IS NULL
		HAVING distance < ?
		ORDER BY distance ASC
	` // HAPUS LIMIT 1

	if err := configs.DB.
		Raw(query, body.Latitude, body.Longitude, body.Latitude, body.Radius).
		Preload("ParentBank").
		Find(&nearbyChildBanks).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to find nearby banks: "+err.Error(), nil, nil)
	}

	// Cari parent bank default (fallback)
	var parentBank models.ParentBank
	configs.DB.First(&parentBank)

	// Format response dengan semua bank terdekat
	type ChildBankWithDistance struct {
		models.ChildBank
		Distance float64 `json:"distance"`
	}

	var childBanksResponse []ChildBankWithDistance
	for _, bank := range nearbyChildBanks {
		distance := helpers.CalculateDistance(body.Latitude, body.Longitude, bank.Latitude, bank.Longitude)
		childBanksResponse = append(childBanksResponse, ChildBankWithDistance{
			ChildBank: bank,
			Distance:  math.Round(distance*100) / 100,
		})
	}

	// Response
	response := map[string]interface{}{
		"user_coordinates": map[string]float64{
			"latitude":  body.Latitude,
			"longitude": body.Longitude,
		},
		"radius_km":            body.Radius,
		"nearby_child_banks":   childBanksResponse, // Sekarang array, bukan single object
		"nearby_banks_count":   len(childBanksResponse),
		"fallback_parent_bank": parentBank,
		"has_nearby_banks":     len(childBanksResponse) > 0,
	}

	// Tentukan message berdasarkan jumlah bank yang ditemukan
	if len(childBanksResponse) > 0 {
		response["message"] = fmt.Sprintf("Ditemukan %d bank pembantu dalam radius %.1fKM", len(childBanksResponse), body.Radius)
		response["bank_type"] = "child_bank"
	} else {
		response["message"] = fmt.Sprintf("Tidak ada bank pembantu dalam radius %.1fKM. Request akan dikirim ke bank induk.", body.Radius)
		response["bank_type"] = "parent_bank"
	}

	return helpers.Response(c, fiber.StatusOK, "Success", "Nearby banks check completed", response, nil)
}

// CreatePickupRequest - Create pickup request dimana user memilih bank (child bank atau parent bank)
func CreatePickupRequest(c *fiber.Ctx) error {
	var body struct {
		UserID       uint    `json:"user_id"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		ChildBankID  *uint   `json:"child_bank_id"`  // Optional - user pilih salah satu
		ParentBankID *uint   `json:"parent_bank_id"` // Optional - user pilih salah satu
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Failed to read body: "+err.Error(), nil, nil)
	}

	// Validasi required fields
	if body.UserID == 0 || body.Latitude == 0 || body.Longitude == 0 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "User ID and coordinates are required", nil, nil)
	}

	// Validasi: user harus pilih salah satu, child bank ATAU parent bank
	if body.ChildBankID == nil && body.ParentBankID == nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Either child_bank_id or parent_bank_id is required", nil, nil)
	}
	if body.ChildBankID != nil && body.ParentBankID != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Cannot specify both child_bank_id and parent_bank_id", nil, nil)
	}

	// Validasi koordinat
	if body.Latitude < -90 || body.Latitude > 90 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid latitude value", nil, nil)
	}
	if body.Longitude < -180 || body.Longitude > 180 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid longitude value", nil, nil)
	}

	// Start transaction
	tx := configs.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Cek user exists
	var user models.User
	if err := tx.First(&user, body.UserID).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Tentukan bank type dan validasi bank exists
	var bankType string
	var selectedBank interface{}

	if body.ChildBankID != nil {
		// User pilih child bank
		var childBank models.ChildBank
		if err := tx.Preload("ParentBank").First(&childBank, *body.ChildBankID).Error; err != nil {
			tx.Rollback()
			return helpers.Response(c, fiber.StatusNotFound, "Failed", "Child bank not found", nil, nil)
		}
		bankType = "child_bank"
		selectedBank = childBank
	} else {
		// User pilih parent bank
		var parentBank models.ParentBank
		if err := tx.First(&parentBank, *body.ParentBankID).Error; err != nil {
			tx.Rollback()
			return helpers.Response(c, fiber.StatusNotFound, "Failed", "Parent bank not found", nil, nil)
		}
		bankType = "parent_bank"
		selectedBank = parentBank
	}

	// Create pickup request
	pickupRequest := models.PickupRequest{
		UserID:       body.UserID,
		ChildBankID:  body.ChildBankID,
		ParentBankID: body.ParentBankID,
		Latitude:     body.Latitude,
		Longitude:    body.Longitude,
		Status:       "pending",
	}

	if err := tx.Create(&pickupRequest).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to create pickup request: "+err.Error(), nil, nil)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Transaction failed: "+err.Error(), nil, nil)
	}

	// Preload relations untuk response
	if err := configs.DB.
		Preload("User").
		Preload("ChildBank").
		Preload("ChildBank.ParentBank").
		Preload("ParentBank").
		First(&pickupRequest, pickupRequest.Id).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to load pickup request data", nil, nil)
	}

	// Response dengan info bank type
	response := map[string]interface{}{
		"pickup_request": pickupRequest,
		"bank_type":      bankType,
		"selected_bank":  selectedBank,
		"message":        getPickupSuccessMessage(bankType, selectedBank),
	}

	return helpers.Response(c, fiber.StatusCreated, "Success", "Pickup request created successfully", response, nil)
}

// Helper function untuk message
func getPickupSuccessMessage(bankType string, bank interface{}) string {
	if bankType == "child_bank" {
		if childBank, ok := bank.(models.ChildBank); ok {
			return fmt.Sprintf("Request penjemputan berhasil dikirim ke bank pembantu %s", childBank.Subdistrict)
		}
	} else {
		if parentBank, ok := bank.(models.ParentBank); ok {
			return fmt.Sprintf("Request penjemputan berhasil dikirim ke bank induk %s", parentBank.District)
		}
	}
	return "Request penjemputan berhasil dikirim"
}

// GetPickupRequests - Get pickup requests with optional filters (user_id untuk child bank, parent_bank_id untuk parent bank)
func GetPickupRequests(c *fiber.Ctx) error {
	userID := c.Query("user_id")
	parentBankID := c.Query("parent_bank_id")

	query := configs.DB.
		Preload("User").
		Preload("User.Role").
		Preload("ChildBank").
		Preload("ParentBank").
		Where("deleted_at IS NULL")

	// Filter by user_id (untuk child bank)
	if userID != "" {
		// Cari user untuk mendapatkan child_bank_id
		var user models.User
		result := configs.DB.
			Select("id", "child_bank_id").
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&user)
		
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
			}
			return helpers.Response(c, 500, "Failed", "Failed to fetch user data", nil, nil)
		}

		// Cek jika user memiliki child_bank_id
		if user.ChildBankID == nil {
			return helpers.Response(c, 404, "Failed", "User does not have associated child bank", nil, nil)
		}

		query = query.Where("child_bank_id = ?", *user.ChildBankID)
	}

	// Filter by parent_bank_id (untuk parent bank)
	if parentBankID != "" {
		query = query.Where("parent_bank_id = ?", parentBankID)
	}

	// Jika tidak ada filter, return semua data
	var pickupRequests []models.PickupRequest
	result := query.Find(&pickupRequests)
	if result.Error != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch pickup requests", nil, nil)
	}

	if len(pickupRequests) == 0 {
		return helpers.Response(c, 404, "Failed", "No pickup requests found", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Pickup requests retrieved successfully", pickupRequests, nil)
}
func UpdatePickupRequestStatus(c *fiber.Ctx) error {
    id := c.Params("id_request")
    status := c.Params("status") // "confirm", "reject", or "complete"

    // Validate status
    if status != "confirm" && status != "reject" && status != "complete" {
        return helpers.Response(c, 400, "Failed", "Invalid status. Use 'confirm', 'reject' or 'complete'", nil, nil)
    }

    var pickupRequest models.PickupRequest
    result := configs.DB.
        Preload("User").
        Preload("User.Role").
        Preload("ChildBank").
        Preload("ParentBank").
        Where("id = ? AND deleted_at IS NULL", id).
        First(&pickupRequest)
    
    if result.Error != nil {
        if result.Error == gorm.ErrRecordNotFound {
            return helpers.Response(c, 404, "Failed", "Pickup request not found", nil, nil)
        }
        return helpers.Response(c, 500, "Failed", "Failed to fetch pickup request", nil, nil)
    }

    targetStatus := status
    message := status

    // Mapping untuk response message yang lebih user-friendly
    if status == "confirm" {
        message = "confirm" // Tetap "confirm" untuk response
    }

    // Cek jika status sudah sama
    if pickupRequest.Status == targetStatus {
        return helpers.Response(c, 400, "Failed", fmt.Sprintf("Pickup request already %sed", message), nil, nil)
    }

    // Validasi status transition
    switch status {
    case "confirm":
        if pickupRequest.Status != "pending" {
            return helpers.Response(c, 400, "Failed", "Can only confirm pending pickup requests", nil, nil)
        }
    case "reject":
        if pickupRequest.Status != "pending" {
            return helpers.Response(c, 400, "Failed", "Can only reject pending pickup requests", nil, nil)
        }
    case "complete":
        if pickupRequest.Status != "confirm" {
            return helpers.Response(c, 400, "Failed", "Can only complete confirmed pickup requests", nil, nil)
        }
    }

    // Update status
    result = configs.DB.Model(&pickupRequest).Updates(map[string]interface{}{
        "status":     targetStatus,
        "updated_at": time.Now(),
    })
    
    if result.Error != nil {
        fmt.Printf("Database error: %v\n", result.Error)
        return helpers.Response(c, 500, "Failed", fmt.Sprintf("Failed to %s pickup request", message), nil, nil)
    }

    if result.RowsAffected == 0 {
        return helpers.Response(c, 500, "Failed", "No rows affected", nil, nil)
    }

    // Reload data untuk response
    var updatedPickupRequest models.PickupRequest
    result = configs.DB.
        Preload("User").
        Preload("User.Role").
        Preload("ChildBank").
        Preload("ParentBank").
        Where("id = ?", id).
        First(&updatedPickupRequest)

    if result.Error != nil {
        return helpers.Response(c, 500, "Failed", "Failed to fetch updated pickup request data", nil, nil)
    }

    // Response message yang konsisten dengan "confirm"
    responseMessage := fmt.Sprintf("Pickup request %sed successfully", message)
    if message == "confirm" {
        responseMessage = "Pickup request confirmed successfully"
    }

    return helpers.Response(c, 200, "Success", responseMessage, updatedPickupRequest, nil)
}