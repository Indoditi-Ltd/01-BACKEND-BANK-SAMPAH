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

func LoginC(c *fiber.Ctx) error {
	// get email and pass from req body
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
 
	if err := c.BodyParser(&body); err != nil {
		return helpers.Response(c, 500, "Failed", "Failed to read body", nil, nil)
	}

	var user models.User
	configs.DB.Preload("Division").Preload("Role").Preload("Plan").First(&user, "email = ?", body.Email)

	if user.Id == 0 {
		return helpers.Response(c, 400, "Failed", "User not found", nil, nil)
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Password wrong!", nil, nil)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Id,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return helpers.Response(c, 400, "Failed", "Invalid to create token", nil, nil)
	}

	return helpers.Response(c, 200, "Success", "Data User found", user, &tokenString)
}
