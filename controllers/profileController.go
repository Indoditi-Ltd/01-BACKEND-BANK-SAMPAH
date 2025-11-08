package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"path/filepath"
	// "strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GetAdminProfile - Get logged in admin profile
func GetAdminProfile(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID, err := helpers.ExtractUserID(c)
	if err != nil {
		return helpers.Response(c, 401, "Failed", "Unauthorized: "+err.Error(), nil, nil)
	}

	var user models.User
	if err := configs.DB.
		Preload("Division").
		Preload("Role").
		Preload("Plan").
		Preload("ParentBank").
		First(&user, userID).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
	}

	// Format response
	profileData := fiber.Map{
		"id":          user.Id,
		"name":        user.Name,
		"email":       user.Email,
		"phone":       user.Phone,
		"address":     user.Address,
		"photo":       user.Photo,
		"division":    user.Division,
		"role":        user.Role,
		"plan":        user.Plan,
		"parent_bank": user.ParentBank,
		"norek":       user.Norek,
		"created_at":  user.CreatedAt,
		"updated_at":  user.UpdatedAt,
	}

	return helpers.Response(c, 200, "Success", "Profile retrieved successfully", profileData, nil)
}
// UpdateAdminProfile - Update admin profile
func UpdateAdminProfile(c *fiber.Ctx) error {
	userID, err := helpers.ExtractUserID(c)
	if err != nil {
		return helpers.Response(c, 401, "Failed", "Unauthorized: "+err.Error(), nil, nil)
	}

	// Parse form data
	var body struct {
		Name    string `form:"name"`
		Phone   string `form:"phone"`
		Address string `form:"address"`
		Email   string `form:"email"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 400, "Failed", "Failed to parse request body", nil, nil)
	}

	// Check if user exists
	var user models.User
	if err := configs.DB.First(&user, uint(userID)).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
	}

	// Handle file upload if provided
	var photoURL string
	file, err := c.FormFile("photo")
	if err == nil {
		// Validate file
		if file.Size > 2<<20 { // 2MB
			return helpers.Response(c, 400, "Failed", "File size too large (max 2MB)", nil, nil)
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedTypes := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		}
		if !allowedTypes[ext] {
			return helpers.Response(c, 400, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
		}

		// Upload new photo to S3
		s3Service := helpers.NewS3Service()
		photoURL, err = s3Service.UploadFile(file, uint(userID), "admin-profiles")
		if err != nil {
			return helpers.Response(c, 500, "Failed", "Failed to upload photo", nil, nil)
		}

		// Delete old photo from S3 if exists
		if user.Photo != "" {
			oldKey := s3Service.ExtractKeyFromURL(user.Photo)
			if oldKey != "" {
				s3Service.DeleteFile(oldKey) // Ignore error for deletion
			}
		}
	}

	// Check if email already exists (if email is being changed)
	if body.Email != "" && body.Email != user.Email {
		var existingUser models.User
		if err := configs.DB.Where("email = ? AND id != ?", body.Email, user.Id).First(&existingUser).Error; err == nil {
			return helpers.Response(c, 400, "Failed", "Email already exists", nil, nil)
		}
	}

	// Prepare update data - HANYA update field yang diisi
	updateData := make(map[string]any)

	// Only update name if provided
	if body.Name != "" {
		updateData["name"] = body.Name
	}

	// Only update phone if provided
	if body.Phone != "" {
		updateData["phone"] = body.Phone
	}

	// Only update address if provided
	if body.Address != "" {
		updateData["address"] = body.Address
	}

	// Only update email if provided and different
	if body.Email != "" && body.Email != user.Email {
		updateData["email"] = body.Email
	}

	// Update photo if new one was uploaded
	if photoURL != "" {
		updateData["photo"] = photoURL
	}

	// Jika tidak ada field yang diupdate, return
	if len(updateData) == 0 {
		return helpers.Response(c, 400, "Failed", "No data to update", nil, nil)
	}

	// Update user in database
	result := configs.DB.Model(&user).Updates(updateData)
	if result.Error != nil {
		// Delete uploaded photo if update fails
		if photoURL != "" {
			s3Service := helpers.NewS3Service()
			newKey := s3Service.ExtractKeyFromURL(photoURL)
			if newKey != "" {
				s3Service.DeleteFile(newKey)
			}
		}
		return helpers.Response(c, 500, "Failed", "Failed to update profile", nil, nil)
	}

	// Get updated user data with relations
	var updatedUser models.User
	configs.DB.
		Preload("Division").
		Preload("Role").
		Preload("Plan").
		Preload("ParentBank").
		First(&updatedUser, user.Id)

	// Format response
	profileData := fiber.Map{
		"id":          updatedUser.Id,
		"name":        updatedUser.Name,
		"email":       updatedUser.Email,
		"phone":       updatedUser.Phone,
		"address":     updatedUser.Address,
		"photo":       updatedUser.Photo,
		"division":    updatedUser.Division,
		"role":        updatedUser.Role,
		"plan":        updatedUser.Plan,
		"parent_bank": updatedUser.ParentBank,
		"norek":       updatedUser.Norek,
		"updated_at":  updatedUser.UpdatedAt,
	}

	return helpers.Response(c, 200, "Success", "Profile updated successfully", profileData, nil)
}

// UploadProfilePhoto - Upload only profile photo (Debug Version)
func UploadProfilePhoto(c *fiber.Ctx) error {
	userID, err := helpers.ExtractUserID(c)
	if err != nil {
		return helpers.Response(c, 401, "Failed", "Unauthorized: "+err.Error(), nil, nil)
	}
	// Parse form file
	file, err := c.FormFile("photo")
	if err != nil {
		fmt.Printf("Error parsing form file: %v\n", err)
		return helpers.Response(c, 400, "Failed", "No file uploaded", nil, nil)
	}
	fmt.Printf("File received: %s, Size: %d bytes\n", file.Filename, file.Size)

	// Validate file size (max 2MB)
	if file.Size > 2<<20 {
		fmt.Printf("File too large: %d bytes\n", file.Size)
		return helpers.Response(c, 400, "Failed", "File size too large (max 2MB)", nil, nil)
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
		fmt.Printf("Invalid file type: %s\n", ext)
		return helpers.Response(c, 400, "Failed", "Invalid file type. Allowed: JPG, JPEG, PNG, GIF, WEBP", nil, nil)
	}
	fmt.Printf("File type validated: %s\n", ext)

	// Convert userID to uint
	// id, err := strconv.ParseUint(userID, 10, 32)
	// if err != nil {
	// 	fmt.Printf("Invalid user ID: %s\n", userID)
	// 	return helpers.Response(c, 400, "Failed", "Invalid user ID", nil, nil)
	// }

	// Get current user data to delete old photo
	var user models.User
	if err := configs.DB.First(&user, uint(userID)).Error; err != nil {
		fmt.Printf("User not found: %d\n", uint(userID))
		return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
	}
	fmt.Printf("User found: %s\n", user.Name)

	// Initialize S3 service - PERBAIKAN DI SINI
	s3Service := helpers.NewS3Service() // services, bukan helpers
	fmt.Printf("S3 service initialized. Bucket: %s\n", s3Service.GetBucket())

	// Upload to S3
	fmt.Printf("Starting S3 upload...\n")
	photoURL, err := s3Service.UploadFile(file, uint(userID), "profiles")
	if err != nil {
		fmt.Printf("S3 upload failed: %v\n", err)
		// Cek kredensial dan konfigurasi S3
		return helpers.Response(c, 500, "Failed", "Failed to upload photo to cloud storage: "+err.Error(), nil, nil)
	}
	fmt.Printf("S3 upload successful. URL: %s\n", photoURL)

	// Delete old photo from storage if exists
	if user.Photo != "" {
		fmt.Printf("Deleting old photo: %s\n", user.Photo)
		oldKey := s3Service.ExtractKeyFromURL(user.Photo)
		if oldKey != "" {
			if err := s3Service.DeleteFile(oldKey); err != nil {
				fmt.Printf("Warning: Failed to delete old file: %v\n", err)
			} else {
				fmt.Printf("Old photo deleted successfully\n")
			}
		}
	}

	// Update user photo in database
	fmt.Printf("Updating database...\n")
	result := configs.DB.Model(&user).Update("photo", photoURL)
	if result.Error != nil {
		fmt.Printf("Database update failed: %v\n", result.Error)
		// Delete uploaded file from storage if database update fails
		newKey := s3Service.ExtractKeyFromURL(photoURL)
		if newKey != "" {
			s3Service.DeleteFile(newKey)
			fmt.Printf("Rollback: Deleted uploaded file from S3\n")
		}
		return helpers.Response(c, 500, "Failed", "Failed to update profile photo", nil, nil)
	}

	fmt.Printf("Profile photo update completed successfully\n")
	return helpers.Response(c, 200, "Success", "Profile photo uploaded successfully", fiber.Map{
		"photo_url": photoURL,
		"user_id":   userID,
	}, nil)
}

// DeleteProfilePhoto - Delete profile photo
func DeleteProfilePhoto(c *fiber.Ctx) error {
	userID, err := helpers.ExtractUserID(c)
	if err != nil {
		return helpers.Response(c, 401, "Failed", "Unauthorized: "+err.Error(), nil, nil)
	}

	var user models.User
	if err := configs.DB.First(&user, userID).Error; err != nil {
		return helpers.Response(c, 404, "Failed", "User not found", nil, nil)
	}

	if user.Photo == "" {
		return helpers.Response(c, 400, "Failed", "User has no photo", nil, nil)
	}

	// Initialize S3 service
	s3Service := helpers.NewS3Service()

	// Extract key and delete from storage
	key := s3Service.ExtractKeyFromURL(user.Photo)
	if key != "" {
		if err := s3Service.DeleteFile(key); err != nil {
			fmt.Println("Warning: Failed to delete file from storage:", err)
		}
	}

	// Update database
	configs.DB.Model(&user).Update("photo", "")

	return helpers.Response(c, 200, "Success", "Profile photo deleted successfully", nil, nil)
}

// TestNEOConnection - Test connection to NEO Object Storage
func TestNEOConnection(c *fiber.Ctx) error {
	s3Service := helpers.NewS3Service()

	fmt.Printf("Testing NEO Object Storage connection...\n")
	fmt.Printf("Bucket: %s\n", s3Service.GetBucket())
	fmt.Printf("Endpoint: %s\n", s3Service.GetEndpoint())
	fmt.Printf("Region: %s\n", configs.GetAWSRegion())

	// Test connection
	err := s3Service.TestConnection()
	if err != nil {
		errorMsg := fmt.Sprintf("NEO Object Storage connection failed: %v", err)
		fmt.Printf("ERROR: %s\n", errorMsg)

		return helpers.Response(c, 500, "Failed", errorMsg, fiber.Map{
			"config": fiber.Map{
				"bucket":     s3Service.GetBucket(),
				"endpoint":   s3Service.GetEndpoint(),
				"region":     configs.GetAWSRegion(),
				"url_format": "https://nos.jkt-1.neo.id/bucket-name/key",
			},
		}, nil)
	}

	successMsg := "NEO Object Storage connection successful"
	fmt.Printf("SUCCESS: %s\n", successMsg)

	// Contoh URL yang akan dihasilkan
	exampleURL := s3Service.GetFileURL("test-folder/test-file.jpg")

	return helpers.Response(c, 200, "Success", successMsg, fiber.Map{
		"bucket":      s3Service.GetBucket(),
		"endpoint":    s3Service.GetEndpoint(),
		"region":      configs.GetAWSRegion(),
		"example_url": exampleURL,
		"url_format":  "https://nos.jkt-1.neo.id/bucket-name/key",
	}, nil)
}
