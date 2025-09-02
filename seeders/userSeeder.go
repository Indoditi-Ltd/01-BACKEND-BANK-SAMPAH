package seeders

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
)

// SeedUsers mengisi tabel users dengan data awal
func SeedUsers() error {
	// Ambil ID role untuk digunakan
	var adminRole, parentRole, childRole, userRole, partnerRole models.Role
	if err := configs.DB.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}
	if err := configs.DB.Where("name = ?", "parent bank").First(&parentRole).Error; err != nil {
		return err
	}

	if err := configs.DB.Where("name = ?", "child bank").First(&childRole).Error; err != nil {
		return err
	}
	if err := configs.DB.Where("name = ?", "user").First(&userRole).Error; err != nil {
		return err
	}
	if err := configs.DB.Where("name = ?", "partner").First(&partnerRole).Error; err != nil {
		return err
	}

	var goldPlan, platinumPlan models.Plan
	if err := configs.DB.Where("name = ?", "Gold").First(&goldPlan).Error; err != nil {
		return err
	}

	if err := configs.DB.Where("name = ?", "Platinum").First(&platinumPlan).Error; err != nil {
		return err
	}

	// Data user dengan RoleID
	users := []models.User{
		{
			Name:    "Admin User",
			Email:   "admin@example.com",
			Phone:   "+6281234567890",
			Address: "123 Admin Street, Jakarta",
			Photo:   "https://example.com/photos/admin.jpg",
			RoleID:  adminRole.ID,
		},
		{
			Name:    "Role Bank Induk",
			Email:   "bankinduk@example.com",
			Phone:   "+6289876543210",
			Address: "456 User Avenue, Bandung",
			Photo:   "https://example.com/photos/user.jpg",
			RoleID:  parentRole.ID,
		},
		{
			Name:    "Role Bank Pembantu",
			Email:   "bankpembantu@example.com",
			Phone:   "+6289876543210",
			Address: "456 User Avenue, Jember",
			Photo:   "https://example.com/photos/user.jpg",
			RoleID:  childRole.ID,
		},
		{
			Name:    "Role End User",
			Email:   "user@example.com",
			Phone:   "+6289876543210",
			Address: "456 User Avenue, Jember",
			Photo:   "https://example.com/photos/user.jpg",
			RoleID:  userRole.ID,
		},
		{
			Name:    "Role Mitra",
			Email:   "mitra@example.com",
			Phone:   "+6289876543210",
			Address: "456 User Avenue, Jember",
			Photo:   "https://example.com/photos/user.jpg",
			RoleID:  partnerRole.ID,
		},
	}

	for i, user := range users {
		var rawPassword string

		// Sesuaikan password default berdasarkan RoleID
		switch user.RoleID {
		case adminRole.ID:
			rawPassword = "admin123"
		case parentRole.ID:
			rawPassword = "parent123"
		case childRole.ID:
			rawPassword = "child123"
		case userRole.ID: 
			rawPassword = "user123"
		case partnerRole.ID:
			rawPassword = "partner123"
		default:
			rawPassword = "default123"
		}

		hashedPassword, err := helpers.HashPassword(rawPassword)
		if err != nil {
			return err
		}
		users[i].Password = hashedPassword

		// Cek duplikasi user
		var existingUser models.User
		if err := configs.DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
			continue // Lewati jika user sudah ada
		}
		if err := configs.DB.Create(&users[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
