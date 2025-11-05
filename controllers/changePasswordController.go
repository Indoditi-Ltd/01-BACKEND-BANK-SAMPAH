package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"fmt"
	"strconv"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// ChangePassword - Change user password
func ChangePassword(c *fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid user ID", nil, nil)
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Invalid request body: "+err.Error(), nil, nil)
	}

	// Validate required fields
	if body.CurrentPassword == "" || body.NewPassword == "" || body.ConfirmPassword == "" {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Current password, new password, and confirm password are required", nil, nil)
	}

	// Check if new password matches confirm password
	if body.NewPassword != body.ConfirmPassword {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "New password and confirm password do not match", nil, nil)
	}

	// Validate password strength
	if err := validatePasswordStrength(body.NewPassword); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", err.Error(), nil, nil)
	}

	// Get user from database
	var user models.User
	if err := configs.DB.First(&user, userID).Error; err != nil {
		return helpers.Response(c, fiber.StatusNotFound, "Failed", "User not found", nil, nil)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.CurrentPassword)); err != nil {
		return helpers.Response(c, fiber.StatusBadRequest, "Failed", "Current password is incorrect", nil, nil)
	}

	// Hash new password
	hashedPassword, err := helpers.HashPassword(body.NewPassword)
	if err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to hash new password", nil, nil)
	}

	// Update password in database
	user.Password = hashedPassword
	if err := configs.DB.Save(&user).Error; err != nil {
		return helpers.Response(c, fiber.StatusInternalServerError, "Failed", "Failed to update password", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Password changed successfully", nil, nil)
}

// validatePasswordStrength - Validate password meets requirements
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must contain at least 8 characters")
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must include at least one upper case letter")
	}
	if !hasLower {
		return fmt.Errorf("password must include at least one lower case letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must include at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must include at least one special character")
	}

	return nil
}