package main

import (
	"log"
	"mentoring_backend/handlers"
	"os"

	"github.com/pocketbase/pocketbase"
)

func main() {
    app := pocketbase.New()

    // Add the afterCreate hook to trigger Lambda for mentee registration
    app.OnRecordAfterCreateRequest().Add(handlers.HandleMenteeRegistration(app))

    // Check for PORT environment variable (default to 8080 if not set)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // Default to 8080 for production
    }

    // Set the port in the root command
    app.RootCmd.SetArgs([]string{"serve", "--http=0.0.0.0:" + port})

    // Start the PocketBase app
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
