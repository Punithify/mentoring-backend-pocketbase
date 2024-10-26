package main

import (
	"log"
	"mentoring_backend/handlers" // Assuming your handler is in this directory

	"github.com/pocketbase/pocketbase"
)

func main() {
    app := pocketbase.New()

    // Add the afterCreate hook to trigger Lambda for mentee registration
    app.OnRecordAfterCreateRequest().Add(handlers.HandleMenteeRegistration(app))

    // Start the PocketBase app
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
