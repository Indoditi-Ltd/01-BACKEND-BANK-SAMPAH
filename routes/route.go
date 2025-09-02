package routes

import (
	"backend-mulungs/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupRoute(app *fiber.App) {
	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	// Group API
	api := app.Group("/api")
	{
		api.Post("/login", controllers.LoginC)
	}
}
