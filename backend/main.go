package main

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samouraiworld/gnomonitoring/backend/internal"
	"github.com/samouraiworld/gnomonitoring/backend/internal/api"
	"github.com/samouraiworld/gnomonitoring/backend/internal/database"
	"github.com/samouraiworld/gnomonitoring/backend/internal/gnovalidator"
	"github.com/samouraiworld/gnomonitoring/backend/internal/govdao"
	"github.com/samouraiworld/gnomonitoring/backend/internal/scheduler"
)

// @title Gno Monitoring Backend API
// @version 1.0.0
// @description Comprehensive blockchain monitoring and alerting system for Gno validators
// @termsOfService http://swagger.io/terms/

// @contact.name Samourai Team
// @contact.url https://github.com/samouraiworld/gnomonitoring
// @contact.email support@samouraiworld.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey DevAuth
// @in header
// @name X-Debug-UserID
// @description Development mode: Use any user ID for testing

// var db *sql.DB

func main() {
	internal.LoadConfig()
	// Initialise les flags
	internal.InitFlags()

	db, err := database.InitDB("./db/webhooks.db")
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ Failed to get underlying SQL DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Database is not reachable: %v", err)
	}

	log.Println("✅ Database connection established successfully")

	go gnovalidator.StartValidatorMonitoring(db) // gnovalidator realtime

	if !*internal.DisableReport {
		go scheduler.InitScheduler(db)
	} else {
		log.Println("⚠️ Daily report scheduler disabled by flag")
	} // for dailyreport

	go govdao.StartGovDAo(db)
	go govdao.StartProposalWatcher(db)

	gnovalidator.Init()                  // init metrics prometheus
	gnovalidator.StartMetricsUpdater(db) // update metrics prometheus / 5 min
	go gnovalidator.StartPrometheusServer(internal.Config.MetricsPort)

	api.StartWebhookAPI(db) //API
	select {}
}
