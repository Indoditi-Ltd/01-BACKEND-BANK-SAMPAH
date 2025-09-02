package helpers

import (
	"github.com/gofiber/fiber/v2"
)

type ResponseWithData struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Token   *string `json:"token"`
	Data    any     `json:"data"`
}

type ResponseWithoutData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Response(c *fiber.Ctx, statusCode int, status string, message string, data any, token *string) error {
	if data != nil {
		return c.Status(statusCode).JSON(ResponseWithData{
			Status:  status,
			Message: message,
			Token:   token,
			Data:    data,
		})
	} else {
		return c.Status(statusCode).JSON(ResponseWithoutData{
			Status:  status,
			Message: message,
		})
	}
}
