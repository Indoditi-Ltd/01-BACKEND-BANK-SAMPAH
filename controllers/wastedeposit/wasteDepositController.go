package wastedeposit

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetAllWasteDeposits mendapatkan semua transaksi setoran sampah
func GetAllWasteDeposits(c *fiber.Ctx) error {
	var wasteDeposits []models.WasteDeposit

	if err := configs.DB.
		Preload("User").
		Preload("Items").
		Preload("Items.ProductWaste").
		Find(&wasteDeposits).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to fetch waste deposits", nil, nil)
	}

	if len(wasteDeposits) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.WasteDeposit{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data Found", wasteDeposits, nil)
}

// GetWasteDepositByID mendapatkan transaksi setoran sampah by ID
func GetWasteDepositByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var wasteDeposit models.WasteDeposit

	if err := configs.DB.
		Preload("User").
		Preload("Items").
		Preload("Items.ProductWaste").
		First(&wasteDeposit, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Waste deposit not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Waste deposit retrieved successfully", wasteDeposit, nil)
}

// GetWasteDepositsByUser mendapatkan transaksi setoran sampah by user ID
func GetWasteDepositsByUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")

	var wasteDeposits []models.WasteDeposit

	if err := configs.DB.
		Preload("User").
		Preload("Items").
		Preload("Items.ProductWaste").
		Where("user_id = ?", userID).
		Find(&wasteDeposits).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to fetch waste deposits", nil, nil)
	}

	if len(wasteDeposits) == 0 {
		return helpers.Response(c, 200, "Success", "Data found", []models.WasteDeposit{}, nil)
	}

	return helpers.Response(c, 200, "Success", "Data Found", wasteDeposits, nil)
}

// CreateWasteDeposit membuat transaksi setoran sampah baru dengan upload S3 dan update balance
func CreateWasteDeposit(c *fiber.Ctx) error {
	// Parse sebagai multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid form data: "+err.Error(), nil, nil)
	}

	// Ambil values dari form
	userIDStr := getFirstValue(form.Value["user_id"])

	// Validate required fields
	if userIDStr == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "User ID is required", nil, nil)
	}

	// Convert userID to uint
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID format", nil, nil)
	}

	// Parse items dari form data
	items, err := parseWasteDepositItems(form)
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", err.Error(), nil, nil)
	}

	// Validasi items tidak boleh kosong
	if len(items) == 0 {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Items cannot be empty", nil, nil)
	}

	// Start transaction
	tx := configs.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Generate reference ID
	referenceID := "WD" + time.Now().Format("20060102150405")

	// Calculate totals
	var totalWeight float64
	var totalPrice int

	// Create waste deposit items
	var depositItems []models.WasteDepositItem

	// Handle file uploads untuk setiap item
	files := form.File["photos"]
	fileIndex := 0

	for _, itemReq := range items {
		// Get product waste data
		var productWaste models.ProductWaste
		if err := tx.First(&productWaste, itemReq.ProductWasteID).Error; err != nil {
			tx.Rollback()
			return helpers.Response(c, fiber.StatusNotFound, "Failed", "Product waste not found: "+strconv.Itoa(int(itemReq.ProductWasteID)), nil, nil)
		}

		// Calculate subtotal menggunakan harga dari product waste
		subTotal := int(itemReq.Weight) * productWaste.Price

		// Handle file upload untuk item ini jika ada
		photoURL := ""
		if fileIndex < len(files) {
			file := files[fileIndex]

			// Validate file size (max 2MB)
			if file.Size > 2<<20 {
				tx.Rollback()
				return helpers.Response(c, fiber.StatusBadRequest, "Failed", "File size too large (max 2MB): "+file.Filename, nil, nil)
			}

			// Validate file type
			allowedTypes := map[string]bool{
				".jpg":  true,
				".jpeg": true,
				".png":  true,
				".gif":  true,
				".webp": true,
			}

			ext := strings.ToLower(filepath.Ext(file.Filename))
			if !allowedTypes[ext] {
				tx.Rollback()
				return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP: "+file.Filename, nil, nil)
			}

			// Upload to S3
			s3Service := helpers.NewS3Service()
			photoURL, err = s3Service.UploadFile(file, 0, "waste-deposit")
			if err != nil {
				tx.Rollback()
				return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to upload image to cloud storage: "+err.Error(), nil, nil)
			}

			fileIndex++
		}

		item := models.WasteDepositItem{
			ProductWasteID: itemReq.ProductWasteID,
			Category:       productWaste.Category,
			Weight:         itemReq.Weight,
			Unit:           itemReq.Unit,
			SubTotal:       subTotal,
			Photo:          photoURL,
		}

		depositItems = append(depositItems, item)
		totalWeight += itemReq.Weight
		totalPrice += subTotal
	}

	// Create waste deposit
	wasteDeposit := models.WasteDeposit{
		UserID:      uint(userID),
		TotalWeight: totalWeight,
		TotalPrice:  totalPrice,
		ReferenceID: referenceID,
		Items:       depositItems,
	}

	if err := tx.Create(&wasteDeposit).Error; err != nil {
		tx.Rollback()

		// Rollback: Hapus file yang sudah diupload ke S3 jika database error
		for _, item := range depositItems {
			if item.Photo != "" {
				s3Service := helpers.NewS3Service()
				imageKey := s3Service.ExtractKeyFromURL(item.Photo)
				if imageKey != "" {
					s3Service.DeleteFile(imageKey)
				}
			}
		}

		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate entry") {
			return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Reference ID already exists", nil, nil)
		}
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", err.Error(), nil, nil)
	}

	// ✅ UPDATE USER BALANCE DALAM TRANSACTION YANG SAMA
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()

		// Rollback: Hapus file S3
		for _, item := range depositItems {
			if item.Photo != "" {
				s3Service := helpers.NewS3Service()
				imageKey := s3Service.ExtractKeyFromURL(item.Photo)
				if imageKey != "" {
					s3Service.DeleteFile(imageKey)
				}
			}
		}

		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Update user balance
	newBalance := user.Balance + totalPrice
	if err := tx.Model(&user).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()

		// Rollback: Hapus file S3
		for _, item := range depositItems {
			if item.Photo != "" {
				s3Service := helpers.NewS3Service()
				imageKey := s3Service.ExtractKeyFromURL(item.Photo)
				if imageKey != "" {
					s3Service.DeleteFile(imageKey)
				}
			}
		}

		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update user balance", nil, nil)
	}

	// ✅ CREATE TRANSACTION RECORD UNTUK RIWAYAT
	transaction := models.Transaction{
		UserID:  uint(userID),
		Balance: totalPrice,
		Type:    "topup",
		Status:  "confirm", // Otomatis confirmed karena dari waste deposit
		Desc:    "Topup dari setoran sampah - Ref: " + referenceID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		// Log error tapi jangan rollback, karena waste deposit dan balance update sudah berhasil
		fmt.Printf("Warning: Failed to create transaction record: %v\n", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()

		// Rollback: Hapus file yang sudah diupload ke S3 jika commit error
		for _, item := range depositItems {
			if item.Photo != "" {
				s3Service := helpers.NewS3Service()
				imageKey := s3Service.ExtractKeyFromURL(item.Photo)
				if imageKey != "" {
					s3Service.DeleteFile(imageKey)
				}
			}
		}

		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Transaction failed", nil, nil)
	}

	// Reload dengan relations termasuk user dengan balance terbaru
	if err := configs.DB.
		Preload("User").
		Preload("Items").
		Preload("Items.ProductWaste").
		First(&wasteDeposit, wasteDeposit.Id).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to load waste deposit data", nil, nil)
	}

	return helpers.Response(c, fiber.StatusOK, "Success", "Waste deposit created successfully and balance updated", wasteDeposit, nil)
}

// DeleteWasteDeposit menghapus transaksi setoran sampah (soft delete) dan kurangi balance
func DeleteWasteDeposit(c *fiber.Ctx) error {
	id := c.Params("id")

	var wasteDeposit models.WasteDeposit
	if err := configs.DB.Preload("Items").Preload("User").First(&wasteDeposit, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Waste deposit not found", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", err.Error(), nil, nil)
	}

	// Start transaction untuk konsistensi
	tx := configs.DB.Begin()

	// ✅ KURANGI BALANCE USER
	var user models.User
	if err := tx.First(&user, wasteDeposit.UserID).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Kurangi balance user
	newBalance := user.Balance - wasteDeposit.TotalPrice
	if newBalance < 0 {
		newBalance = 0 // Prevent negative balance
	}

	if err := tx.Model(&user).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update user balance", nil, nil)
	}

	// ✅ CREATE TRANSACTION RECORD UNTUK RIWAYAT (WITHDRAWAL)
	transaction := models.Transaction{
		UserID:  wasteDeposit.UserID,
		Balance: -wasteDeposit.TotalPrice, // Negative untuk withdrawal
		Type:    "withdraw",
		Status:  "confirm",
		Desc:    "Refund dari penghapusan setoran sampah - Ref: " + wasteDeposit.ReferenceID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		fmt.Printf("Warning: Failed to create transaction record: %v\n", err)
	}

	// Hapus file S3 yang terkait sebelum soft delete
	for _, item := range wasteDeposit.Items {
		if item.Photo != "" {
			s3Service := helpers.NewS3Service()
			imageKey := s3Service.ExtractKeyFromURL(item.Photo)
			if imageKey != "" {
				// Hapus file dari S3
				if err := s3Service.DeleteFile(imageKey); err != nil {
					// Log error tapi lanjutkan proses delete
					fmt.Printf("Failed to delete S3 file: %s, error: %v\n", imageKey, err)
				}
			}
		}
	}

	// Soft delete waste deposit
	if err := tx.Delete(&wasteDeposit).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Failed to delete waste deposit", nil, nil)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Transaction failed", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Waste deposit deleted successfully and balance adjusted", nil, nil)
}

// Helper function untuk parse items dari form data
func parseWasteDepositItems(form *multipart.Form) ([]WasteDepositItemRequest, error) {
	var items []WasteDepositItemRequest

	// Extract values dari form (menggunakan array notation)
	productWasteIDs := form.Value["product_waste_id[]"]
	weights := form.Value["weight[]"]
	units := form.Value["unit[]"]

	// Validasi jumlah items konsisten
	if len(productWasteIDs) != len(weights) || len(weights) != len(units) {
		return nil, fmt.Errorf("inconsistent number of items in form data")
	}

	// Create items slice
	for i := 0; i < len(productWasteIDs); i++ {
		productID, err := strconv.ParseUint(productWasteIDs[i], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid product_waste_id at index %d", i)
		}

		weight, err := strconv.ParseFloat(weights[i], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid weight at index %d", i)
		}

		item := WasteDepositItemRequest{
			ProductWasteID: uint(productID),
			Weight:         weight,
			Unit:           units[i],
		}
		items = append(items, item)
	}

	return items, nil
}

// Helper function untuk ambil first value dari array
func getFirstValue(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// WasteDepositItemRequest struct untuk request item
type WasteDepositItemRequest struct {
	ProductWasteID uint    `json:"product_waste_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
}
