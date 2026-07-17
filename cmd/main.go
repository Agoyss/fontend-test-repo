package main

import (
	"frontend-test/database"
	"frontend-test/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	database.LoadDotEnv(".env")
	db, err := database.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Force a clean re-migrate when RESET_DB=true (drops tables + tracking, then re-applies).
	if database.Env("RESET_DB", "") == "true" {
		if err := database.Reset(db); err != nil {
			panic(err)
		}
	}

	if err := database.Migrate(db, "migrations"); err != nil {
		panic(err)
	}

	r := gin.Default()
	routes.RegisterRoutes(r, db)

	port := database.Env("PORT", "8080")
	r.Run(":" + port)
}
