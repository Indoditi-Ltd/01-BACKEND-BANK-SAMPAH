package main

import (
	"backend-mulungs/configs"
	"backend-mulungs/initializers"
	"backend-mulungs/routes"
	"backend-mulungs/seeders"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func init() {
	initializers.LoadEnvVarables()
	configs.ConnectDB()
	configs.DatabaseSync()

	// Jalankan seeder
	if err := seeders.SeedAll(); err != nil {
		panic("Failed to seed database: " + err.Error())
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Ganti dengan domain frontend
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "*",
		// AllowCredentials: true,
	}))

	routes.SetupRoute(app)

	app.Listen(":" + port)
}
