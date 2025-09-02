package helpers

import (
	"github.com/gofiber/fiber/v2"
)

type ResponseWithData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ResponseWithoutData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Response(c *fiber.Ctx, statusCode int, status string, message string, data any) error {
	if data != nil {
		return c.Status(statusCode).JSON(ResponseWithData{
			Status:  status,
			Message: message,
			Data:    data,
		})
	} else {
		return c.Status(statusCode).JSON(ResponseWithoutData{
			Status:  status,
			Message: message,
		})
	}
}
