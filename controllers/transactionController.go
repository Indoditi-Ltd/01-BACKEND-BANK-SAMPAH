package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func TransactionCreateTopUp(c *fiber.Ctx) error {
	var body struct {
		UserID  uint   `json:"user_id"`
		Balance int    `json:"balance"`
		AdminID uint   `json:"admin_id"`
		Desc    string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	transaction := models.Transaction{
		UserID:  body.UserID,
		Balance: body.Balance,
		Status:  "pending",
		Desc:    body.Desc,
		Type:    "topup",
	}

	if err := configs.DB.Create(&transaction).Error; err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to create transaction", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Transaction create successfully", transaction, nil)
}
func TransactionCreateWithdraw(c *fiber.Ctx) error {
	var body struct {
		UserID  uint   `json:"user_id"`
		Balance int    `json:"balance"`
		Desc    string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid request body", nil, nil)
	}

	// Mulai transaction database
	tx := configs.DB.Begin()
	if tx.Error != nil {
		return helpers.Response(c, 500, "Failed", "Gagal memulai transaksi database", nil, nil)
	}

	// Cek saldo user dengan lock untuk menghindari race condition
	var user models.User
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, body.UserID).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "User tidak ditemukan", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Gagal mengambil data user", nil, nil)
	}

	// Validasi saldo mencukupi
	if user.Balance < body.Balance {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Saldo tidak mencukupi", nil, nil)
	}

	// Kurangi saldo user
	err = tx.Model(&models.User{}).Where("id = ?", body.UserID).
		Update("balance", gorm.Expr("balance - ?", body.Balance)).Error
	if err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal mengurangi saldo user", nil, nil)
	}

	// Buat transaksi withdraw
	transaction := models.Transaction{
		UserID:  body.UserID,
		Balance: body.Balance,
		Status:  "pending",
		Desc:    body.Desc,
		Type:    "withdraw",
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Failed to create transaction", nil, nil)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal menyimpan transaksi", nil, nil)
	}

	// Reload transaksi dengan data user terbaru
	configs.DB.Preload("User").First(&transaction, transaction.Id)

	res := models.TransactionCreateResponse{
		UserID:  transaction.UserID,
		Balance: transaction.Balance,
		Type:    transaction.Type,
		Status:  transaction.Status,
		Desc:    transaction.Desc,
	}

	return helpers.Response(c, 200, "Success", "Withdraw request created successfully", res, nil)
}

func TransactionAllTopUp(c *fiber.Ctx) error {
	var req struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Search    string `query:"search"`
		Status    string `query:"status"`
		Page      int    `query:"page"`
		Limit     int    `query:"limit"`
	}

	// Parse query parameters
	if err := c.QueryParser(&req); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to parse query parameters", nil, nil)
	}

	// Set default values
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit

	// Build query
	query := configs.DB.Model(&models.Transaction{}).
		Preload("User").
		Preload("User.Plan").
		Preload("Admin").
		Where("type = ?", "topup")

	// Apply date filter
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			query = query.Where("DATE(created_at) >= ?", startDate.Format("2006-01-02"))
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			query = query.Where("DATE(created_at) <= ?", endDate.Format("2006-01-02"))
		}
	}

	// Apply status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Apply search filter (nama/email user) - CARA AMAN
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"

		// Cari user yang sesuai dengan search pattern
		var userIDs []uint
		configs.DB.Model(&models.User{}).
			Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
			Pluck("id", &userIDs)

		// Jika ada user yang ditemukan, filter transaksi berdasarkan user_id
		if len(userIDs) > 0 {
			query = query.Where("user_id IN (?)", userIDs)
		} else {
			// Jika tidak ada user yang cocok, return empty result
			data := map[string]any{
				"transactions": []models.Transaction{},
				"meta": map[string]any{
					"page":  req.Page,
					"limit": req.Limit,
					"total": 0,
					"pages": 0,
				},
			}
			return helpers.Response(c, 200, "Success", "Data found", data, nil)
		}
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get transactions dengan pagination
	var transactions []models.Transaction
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&transactions).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch topup transactions", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]any{
		"transactions": transactions,
		"meta": map[string]any{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Data found", data, nil)
}

func TransactionAllWithdraw(c *fiber.Ctx) error {
	var req struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Search    string `query:"search"`
		Status    string `query:"status"`
		Page      int    `query:"page"`
		Limit     int    `query:"limit"`
	}

	// Parse query parameters
	if err := c.QueryParser(&req); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to parse query parameters", nil, nil)
	}

	// Set default values
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	offset := (req.Page - 1) * req.Limit

	// Build query
	query := configs.DB.Model(&models.Transaction{}).
		Preload("User").
		Preload("User.Plan").
		Preload("Admin").
		Where("type = ?", "withdraw")

	// Apply date filter
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			query = query.Where("DATE(created_at) >= ?", startDate.Format("2006-01-02"))
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			query = query.Where("DATE(created_at) <= ?", endDate.Format("2006-01-02"))
		}
	}

	// Apply status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Apply search filter (nama/email user) - PERBAIKAN DI SINI
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"

		// Cari user yang sesuai dengan search pattern
		var userIDs []uint
		configs.DB.Model(&models.User{}).
			Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
			Pluck("id", &userIDs)

		// Jika ada user yang ditemukan, filter transaksi berdasarkan user_id
		if len(userIDs) > 0 {
			query = query.Where("user_id IN (?)", userIDs)
		} else {
			// Jika tidak ada user yang cocok, return empty result
			data := map[string]any{
				"transactions": []models.Transaction{},
				"meta": map[string]any{
					"page":  req.Page,
					"limit": req.Limit,
					"total": 0,
					"pages": 0,
				},
			}
			return helpers.Response(c, 200, "Success", "Data found", data, nil)
		}
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	// Get transactions dengan pagination
	var transactions []models.Transaction
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&transactions).Error

	if err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to fetch withdrawal transactions", nil, nil)
	}

	// Format response dengan meta di dalam data
	data := map[string]any{
		"transactions": transactions,
		"meta": map[string]any{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	return helpers.Response(c, 200, "Success", "Data found", data, nil)
}

// TransactionDetailResponse struct untuk response JSON
type TransactionDetailResponse struct {
	NamaUser   string `json:"nama_user"`
	NoRekening string `json:"no_rekening"`
	Email      string `json:"email"`
	Tanggal    string `json:"tanggal"`
	TotalTopUp string `json:"total_top_up"`
}

// GetTransactionDetailHandler menampilkan detail transaksi transfer/topup
func GetTransactionDetailHandler(c *fiber.Ctx) error {
	// Ambil ID dari parameter URL
	idParam := c.Params("id")
	if idParam == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID transaksi tidak ditemukan",
		})
	}

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID transaksi tidak valid",
		})
	}

	// Inisialisasi variabel untuk menyimpan data
	var transaction models.Transaction

	// Query ke database dengan preload User
	err = configs.DB.Preload("User").First(&transaction, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error": "Transaksi tidak ditemukan",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal mengambil data transaksi",
		})
	}

	// Format tanggal
	tanggal := transaction.CreatedAt.Format("02 January 2006")

	// Format total top up dengan Rp dan titik ribuan
	totalTopUp := fmt.Sprintf("Rp %s", helpers.FormatCurrencyTransaction(transaction.Balance))

	// Jika ingin ambil No Rekening dari ChildBank (lebih kompleks), bisa tambahkan preload
	// Tapi karena di model User sudah ada `Norek`, kita pakai itu dulu
	noRekening := ""
	if transaction.User.Norek != nil {
		noRekening = fmt.Sprintf("%d", *transaction.User.Norek)
	} else {
		noRekening = "Tidak tersedia"
	}

	// Siapkan response
	response := TransactionDetailResponse{
		NamaUser:   transaction.User.Name,
		NoRekening: noRekening,
		Email:      transaction.User.Email,
		Tanggal:    tanggal,
		TotalTopUp: totalTopUp,
	}

	return c.JSON(response)
}

// ConfirmTransactionHandler mengubah status transaksi menjadi "confirm"
func ConfirmTransactionHandler(c *fiber.Ctx) error {
	// Ambil ID dari parameter URL
	idParam := c.Params("id")
	if idParam == "" {
		return helpers.Response(c, 400, "Failed", "ID transaksi tidak ditemukan", nil, nil)
	}

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "ID transaksi tidak valid", nil, nil)
	}

	// Mulai transaction database
	tx := configs.DB.Begin()
	if tx.Error != nil {
		return helpers.Response(c, 500, "Failed", "Gagal memulai transaksi database", nil, nil)
	}

	// Cari transaksi dengan lock untuk menghindari race condition
	var transaction models.Transaction
	err = tx.Set("gorm:query_option", "FOR UPDATE").Preload("User").First(&transaction, id).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Transaksi tidak ditemukan", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Gagal mengambil data transaksi", nil, nil)
	}

	// Validasi: hanya bisa dikonfirmasi jika status masih "pending"
	if transaction.Status != "pending" {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Transaksi sudah tidak dalam status 'pending'", nil, nil)
	}

	// Update status transaksi
	transaction.Status = "confirm"
	// Set admin ID jika diperlukan
	// adminID, err := GetAdminIDFromContext(c)
	// if err == nil {
	// 	transaction.AdminID = &adminID
	// }

	// Simpan perubahan transaksi
	err = tx.Save(&transaction).Error
	if err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal memperbarui status transaksi", nil, nil)
	}

	// Untuk topup: tambahkan balance user
	if transaction.Type == "topup" {
		err = tx.Model(&models.User{}).Where("id = ?", transaction.UserID).
			Update("balance", gorm.Expr("balance + ?", transaction.Balance)).Error
		if err != nil {
			tx.Rollback()
			return helpers.Response(c, 500, "Failed", "Gagal menambah balance user", nil, nil)
		}
	}
	// Untuk withdraw: tidak perlu melakukan apa-apa karena saldo sudah dipotong saat create

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal menyimpan perubahan", nil, nil)
	}

	// Reload transaksi dengan data terbaru
	configs.DB.Preload("User").First(&transaction, id)

	var message string
	if transaction.Type == "topup" {
		message = "Topup berhasil dikonfirmasi dan balance user telah ditambahkan"
	} else {
		message = "Withdraw berhasil dikonfirmasi"
	}

	return helpers.Response(c, 200, "Success", message, transaction, nil)
}

// RejectTransactionHandler mengubah status transaksi menjadi "reject"
func RejectTransactionHandler(c *fiber.Ctx) error {
	// Ambil ID dari parameter URL
	idParam := c.Params("id")
	if idParam == "" {
		return helpers.Response(c, 400, "Failed", "ID transaksi tidak ditemukan", nil, nil)
	}

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return helpers.Response(c, 400, "Failed", "ID transaksi tidak valid", nil, nil)
	}

	// Mulai transaction database
	tx := configs.DB.Begin()
	if tx.Error != nil {
		return helpers.Response(c, 500, "Failed", "Gagal memulai transaksi database", nil, nil)
	}

	// Cari transaksi dengan lock untuk menghindari race condition
	var transaction models.Transaction
	err = tx.Set("gorm:query_option", "FOR UPDATE").Preload("User").First(&transaction, id).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return helpers.Response(c, 404, "Failed", "Transaksi tidak ditemukan", nil, nil)
		}
		return helpers.Response(c, 500, "Failed", "Gagal mengambil data transaksi", nil, nil)
	}

	// Validasi: hanya bisa direject jika status masih "pending"
	if transaction.Status != "pending" {
		tx.Rollback()
		return helpers.Response(c, 400, "Failed", "Transaksi sudah tidak dalam status 'pending'", nil, nil)
	}

	// Update status transaksi
	transaction.Status = "reject"
	// Set admin ID jika diperlukan (ambil dari JWT atau context)
	// adminID, err := GetAdminIDFromContext(c)
	// if err == nil {
	// 	transaction.AdminID = &adminID
	// }

	// Kembalikan balance untuk transaksi withdraw yang direject
	if transaction.Type == "withdraw" {
		// Untuk withdraw yang direject: kembalikan balance ke user
		err = tx.Model(&models.User{}).Where("id = ?", transaction.UserID).
			Update("balance", gorm.Expr("balance + ?", transaction.Balance)).Error
		if err != nil {
			tx.Rollback()
			return helpers.Response(c, 500, "Failed", "Gagal mengembalikan balance user", nil, nil)
		}
	}

	// Simpan perubahan transaksi
	err = tx.Save(&transaction).Error
	if err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal memperbarui status transaksi", nil, nil)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return helpers.Response(c, 500, "Failed", "Gagal menyimpan perubahan", nil, nil)
	}

	// Reload transaksi dengan data terbaru
	configs.DB.Preload("User").First(&transaction, id)

	return helpers.Response(c, 200, "Success", "Transaksi berhasil ditolak", transaction, nil)
}