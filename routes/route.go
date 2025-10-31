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
		api.Post("/register-user", controllers.RegisterUser)
		api.Post("/callback", controllers.CallbackPrepaid)

		app.Use(middleware.RequireAuth)

		api.Get("/profile", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"message": "Success",
			})
		})

		// for website
		api.Get("/list-topup", controllers.TransactionAllTopUp)
		api.Get("/list-withdraw", controllers.TransactionAllWithdraw)
		api.Get("/roles", controllers.ListRoleC)

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

		userAdmin := api.Group("/admins")
		{
			userAdmin.Post("/", controllers.CreateAdmin)
			userAdmin.Get("/", controllers.GetAllAdmin)
			userAdmin.Get("/:id", controllers.GetAdminByID)
			userAdmin.Put("/:id", controllers.UpdateUserAdmin)
			userAdmin.Delete("/:id", controllers.DeleteUserAdmin)
		}

		ppob := api.Group("/ppob")
		{
			ppob.Get("/prepaid/:type?", controllers.GetListPrepaid)
			ppob.Post("/prepaid/topup", controllers.TopupPrepaid)
			ppob.Post("/postpaid/inquiry", controllers.PostpaidInquiry)
			ppob.Get("/postpaid/:type?", controllers.GetListPostpaid)
			ppob.Get("/postpaid/:type/:province", controllers.GetListPostpaid)
			ppob.Post("/margin", controllers.CreateMargin)
			
			ppob.Get("/history", controllers.GetHistoryByRefID)
		}

		region := api.Group("/region")
		{
			region.Get("/province", controllers.RegionProvince)
			region.Get("/district", controllers.RegionDistrict)
			region.Get("/subdistrict", controllers.RegionSubDistrict)
			region.Get("/village", controllers.RegionVillage)
		}

		dash := api.Group("/dashboard")
		{
			dash.Get("/admin", controllers.DashboardController)
		}
	}
}
