package routes

import (
	"backend-mulungs/controllers"
	"backend-mulungs/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoute(app *fiber.App) {
	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Project Connect",
		})
	})

	// Group API
	api := app.Group("/api")
	{
		api.Post("/login", controllers.LoginC)

		app.Use(middleware.RequireAuth)

		api.Get("/profile", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"message": "Success",
			})
		})

		api.Post("/create-topup", controllers.TransactionCreateTopUp)
		api.Post("/create-withdraw", controllers.TransactionCreateWithdraw)
		api.Get("/list-topup", controllers.TransactionAllTopUp)
		api.Get("/list-withdraw", controllers.TransactionAllWithdraw)
	}
}
