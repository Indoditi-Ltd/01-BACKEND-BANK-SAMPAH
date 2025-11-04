package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)
func RequireAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Missing token",
		})
	}

	// cek token valid
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	_, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "Failed",
			"message": "Invalid token",
		})
	}

	// tanpa c.Locals
	return c.Next()
}

