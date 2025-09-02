package controllers

import (
	"backend-mulungs/configs"
	"backend-mulungs/helpers"
	"backend-mulungs/models"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// func RegisterUser(c *gin.Context) {
// 	// get email and pass of req body
// 	var body struct {
// 		Email    string
// 		Password string
// 	}

// 	if c.Bind(&body) != nil {
// 		helpers.Response(c, 500, "Failed", "Failed to read body", nil)
// 		return
// 	}

// 	hashPassword, err := helpers.HashPassword(body.Password)
// 	if err != nil {
// 		helpers.Response(c, 500, "Failed", "Failed to hash password: "+err.Error(), nil)
// 		return
// 	}

// }

func LoginC(c *fiber.Ctx) error {
	// get email and pass from req body
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to read body", nil)
	}

	var user models.User
	configs.DB.Preload("Role").First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		return helpers.Response(c, 400, "Failed", "User not found", nil)
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Password wrong!", nil)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid to create token", nil)
	}

	// Set cookie in Fiber
	c.Cookie(&fiber.Cookie{
		Name:     "Authorize",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return helpers.Response(c, 200, "Success", "Data user found", user)
}
