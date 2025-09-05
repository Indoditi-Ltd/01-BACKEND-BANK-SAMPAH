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

		// for website
		api.Get("/list-topup", controllers.TransactionAllTopUp)
		api.Get("/list-withdraw", controllers.TransactionAllWithdraw)

		// for role parent bank, child bank, mitra, end user
		api.Post("/create-topup", controllers.TransactionCreateTopUp)
		api.Post("/create-withdraw", controllers.TransactionCreateWithdraw)

		parentBank := api.Group("/parent-bank")
		{
			// Website Admin
			parentBank.Get("/", controllers.GetPrentBank)
			parentBank.Post("/", controllers.CreateParentBank)
			parentBank.Put("/:id", controllers.UpdateParentBank)
			parentBank.Delete("/:id", controllers.DeleteParentBank)

			// Mobile Parent Bank
			parentBank.Get("/:id", controllers.GetParentBankID)
		}

		childBank := api.Group("/child-bank")
		{
			childBank.Post("/", controllers.CreateChildBank)
			childBank.Get("/", controllers.GetAllChildBanks)
			childBank.Get("/:id", controllers.GetChildBankById)
			childBank.Put("/:id", controllers.UpdateChildBank)
			childBank.Delete("/:id", controllers.DeleteChildBank)

		}
	}
}
