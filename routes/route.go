package routes

import (
	"backend-mulungs/controllers"
	"backend-mulungs/controllers/donation"
	pickuprequest "backend-mulungs/controllers/pickupRequest"
	"backend-mulungs/controllers/wastedeposit"
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
		api.Post("/register-user-child-bank", controllers.RegisterUserChildBank)
		api.Post("/callback", controllers.CallbackPrepaid)
		api.Get("/testBucket", controllers.TestNEOConnection)

		app.Use(middleware.RequireAuth)

		// api.Get("/profile", func(c *fiber.Ctx) error {
		// 	return c.JSON(fiber.Map{
		// 		"message": "Success",
		// 	})
		// })

		// Scan Barcode User
		api.Post("/scan-user", controllers.ScanBarcodeUser)

		// for website
		api.Get("/list-topup", controllers.TransactionAllTopUp)
		api.Get("/list-withdraw", controllers.TransactionAllWithdraw)
		api.Get("/roles", controllers.ListRoleC)
		api.Get("/division", controllers.DivisiUserController)
		api.Put("/:id/change-password", controllers.ChangePassword)

		// for role parent bank, child bank, mitra, end user
		api.Post("/create-topup", controllers.TransactionCreateTopUp)
		api.Post("/create-withdraw", controllers.TransactionCreateWithdraw)

		transaction := api.Group("/transactions")
		{
			transaction.Get("/:id", controllers.GetTransactionDetailHandler)
			transaction.Put("/:id/confirm", controllers.ConfirmTransactionHandler)
			transaction.Put("/:id/reject", controllers.RejectTransactionHandler)
		}

		profileGroup := api.Group("/profile")
		{
			profileGroup.Get("/", controllers.GetAdminProfile)
			profileGroup.Put("/", controllers.UpdateAdminProfile)
			profileGroup.Post("/photo", controllers.UploadProfilePhoto)
			profileGroup.Delete("/photo", controllers.DeleteProfilePhoto)
		}

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
			childBank.Get("/by-id/:id", controllers.GetUserChildBankByIDC)
			childBank.Put("/:id", controllers.UpdateChildBank)
			childBank.Delete("/:id", controllers.DeleteChildBank)

		}

		userChildBank := api.Group("/user-childbank")
		{
			userChildBank.Get("/", controllers.GetAllUsersChildBank)
			userChildBank.Post("/", controllers.CreateUserChildBank)
			userChildBank.Put("/:id", controllers.UpdateUserChildBank)
			userChildBank.Delete("/:id", controllers.DeleteUserChildBank)
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
			ppob.Post("/postpaid/payment", controllers.PaymentPostpaid)
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

		userGroup := api.Group("/parent-bank-users")
		{
			userGroup.Get("/", controllers.GetUsersParentBank)      // Get all users with filters
			userGroup.Get("/reset-filter", controllers.ResetFilter) // Reset filter
			userGroup.Get("/list-parent", controllers.GetParentBanksDropdown)
			userGroup.Post("/", controllers.CreateUserBankInduk) // Create user bank induk
			userGroup.Get("/:id", controllers.GetUserByID)       // Get user by ID
			userGroup.Put("/:id", controllers.UpdateUser)        // Update user
			userGroup.Delete("/:id", controllers.DeleteUser)     // Delete user
		}

		marketings := api.Group("/marketing")
		{
			marketings.Get("/", controllers.GetMarketingList)      // Get dengan filter
			marketings.Post("/", controllers.CreateMarketing)      // Create new
			marketings.Put("/:id", controllers.UpdateMarketing)    // Update
			marketings.Delete("/:id", controllers.DeleteMarketing) // Delete
		}

		wasteGroup := api.Group("/waste")
		{
			wasteGroup.Get("/total", controllers.GetTotalWaste)                // Get total weight
			wasteGroup.Get("/products", controllers.GetProductWasteList)       // Get list dengan filter
			wasteGroup.Post("/products", controllers.CreateProductWaste)       // Create new
			wasteGroup.Put("/products/:id", controllers.UpdateProductWaste)    // Update
			wasteGroup.Delete("/products/:id", controllers.DeleteProductWaste) // Delete
		}

		donationGroup := api.Group("/donations")
		{
			donationGroup.Get("/", controllers.GetDonationList)
			donationGroup.Get("/all", controllers.GetDonationListAll)
			donationGroup.Get("/:id", controllers.GetDonationByID)
			donationGroup.Post("/", controllers.CreateDonation)
			donationGroup.Put("/:id", controllers.UpdateDonation)
			donationGroup.Delete("/:id", controllers.DeleteDonation)
			donationGroup.Post("/:user_id/donate", donation.CreateDonation)                  // POST /donations/:user_id/donate
			donationGroup.Get("/:user_id/history", donation.GetUserDonations)                // GET /donations/:user_id/history
			donationGroup.Post("/:donation_id/complete/:user_id", donation.CompleteDonation) // POST /donations/:donation_id/complete/:user_id   // POST /donations/:id/complete
		}

		wasteDepositGroup := api.Group("/waste-deposits")
		{
			wasteDepositGroup.Get("/", wastedeposit.GetAllWasteDeposits)                                    // Get all waste deposits
			wasteDepositGroup.Get("/:id", wastedeposit.GetWasteDepositByID)                                 // Get by ID
			wasteDepositGroup.Get("/user/:user_id", wastedeposit.GetWasteDepositsByUser)                    // Get by user ID
			wasteDepositGroup.Get("/childbank/:child_bank_id", wastedeposit.GetWasteDepositsByChildBank)    // Get by user ID
			wasteDepositGroup.Get("/parentbank/:parent_bank_id", wastedeposit.GetWasteDepositsByParentBank) // Get by user ID
			wasteDepositGroup.Post("/", wastedeposit.CreateWasteDeposit)                                    // Create new waste deposit
			wasteDepositGroup.Delete("/:id", wastedeposit.DeleteWasteDeposit)                               // Delete waste deposit
		}

		requestPickup := api.Group("/pickup")
		{
			requestPickup.Post("/check-distance", pickuprequest.CheckNearbyBanks)
			requestPickup.Post("/requests", pickuprequest.CreatePickupRequest)
			requestPickup.Get("/list-requests", pickuprequest.GetPickupRequests)
			requestPickup.Put("/requests/:id_request/:status", pickuprequest.UpdatePickupRequestStatus)
		}
	}
}
