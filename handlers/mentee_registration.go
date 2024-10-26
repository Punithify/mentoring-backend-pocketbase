package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// MaxMenteesPerMentor is the limit of mentees a mentor can have
const MaxMenteesPerMentor = 10

// FindAvailableMentor finds an available mentor based on your logic
func FindAvailableMentor(app *pocketbase.PocketBase) (*models.Record, error) {
    expr := dbx.NewExp("role = 'mentor'")

    // Query the database for mentors
    mentors, err := app.Dao().FindRecordsByExpr("users", expr)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch available mentors: %w", err)
    }

    if len(mentors) == 0 {
        return nil, fmt.Errorf("no available mentors found")
    }

    for _, mentor := range mentors {
        // Check if the mentor has less than the maximum number of mentees
        menteeCount, err := GetMenteeCountForMentor(app, mentor.Id)
        if err != nil {
            log.Printf("Error fetching mentee count for mentor %s: %v", mentor.Id, err)
            continue
        }

        if menteeCount < MaxMenteesPerMentor {
            // Found a mentor with available capacity
            return mentor, nil
        }
    }

    return nil, fmt.Errorf("no mentors with available capacity found")
}

// GetMenteeCountForMentor returns the number of mentees assigned to a given mentor
func GetMenteeCountForMentor(app *pocketbase.PocketBase, mentorID string) (int, error) {
    expr := dbx.NewExp("mentor_id = {:mentorID}", dbx.Params{"mentorID": mentorID})

    // Query the allocations collection for the mentor record
    allocations, err := app.Dao().FindRecordsByExpr("allocations", expr)
    if err != nil {
        return 0, fmt.Errorf("failed to fetch allocations for mentor %s: %w", mentorID, err)
    }

    if len(allocations) == 0 {
        return 0, nil
    }

    // Get the first allocation record (since there should be one per mentor)
    allocation := allocations[0]

    // Get the mentee_ids array and return its length
    menteeIDs := allocation.GetStringSlice("mentee_ids")
    return len(menteeIDs), nil
}

// HandleMenteeRegistration allocates a mentor to the mentee when they register
func HandleMenteeRegistration(app *pocketbase.PocketBase) func(e *core.RecordCreateEvent) error {
    return func(e *core.RecordCreateEvent) error {
        // Check if the newly created record is a mentee
        if e.Record.Collection().Name == "users" && e.Record.GetString("role") == "mentee" {
            menteeID := e.Record.Id // Get the newly registered mentee ID

            // Find an available mentor
            mentor, err := FindAvailableMentor(app)
            if err != nil {
                log.Println("Error finding available mentor:", err)
                return err
            }

            // Add the mentee to the mentor's allocation
            err = AddMenteeToAllocation(app, mentor.Id, menteeID)
            if err != nil {
                log.Println("Error adding mentee to allocation:", err)
                return err
            }

            log.Printf("Successfully allocated mentor %s to mentee %s", mentor.Id, menteeID)
        }
        return nil
    }
}

// AddMenteeToAllocation adds a new mentee to the mentor's mentee_ids array
func AddMenteeToAllocation(app *pocketbase.PocketBase, mentorID string, menteeID string) error {
    expr := dbx.NewExp("mentor_id = {:mentorID}", dbx.Params{"mentorID": mentorID})

    // Retrieve the allocation record for the mentor
    allocations, err := app.Dao().FindRecordsByExpr("allocations", expr)
    if err != nil {
        return fmt.Errorf("failed to retrieve allocation for mentor %s: %w", mentorID, err)
    }

    var allocation *models.Record
    if len(allocations) > 0 {
        // Use the existing allocation record
        allocation = allocations[0]
    } else {
        // Create a new allocation record if it doesn't exist
        collection, err := app.Dao().FindCollectionByNameOrId("allocations")
        if err != nil {
            return fmt.Errorf("failed to retrieve allocations collection: %w", err)
        }
        allocation = models.NewRecord(collection)
        allocation.Set("mentor_id", mentorID)
        allocation.Set("allocated_on", time.Now().UTC().Format(time.RFC3339))
        allocation.Set("status", "active")
        allocation.Set("is_assigned", true)
    }

    // Get the existing mentee_ids array or initialize if not found
    menteeIDs := allocation.GetStringSlice("mentee_ids")
    if menteeIDs == nil {
        menteeIDs = []string{} // Initialize if it's nil
    }

    // Debug: Log current menteeIDs before appending
    log.Printf("Before appending, mentee_ids: %v", menteeIDs)

    // Check if menteeID is already present to avoid duplicates
    for _, id := range menteeIDs {
        if id == menteeID {
            log.Printf("Mentee %s is already assigned to mentor %s", menteeID, mentorID)
            return nil // No need to append if mentee is already assigned
        }
    }

    // Append the new mentee ID to the mentee_ids array
    menteeIDs = append(menteeIDs, menteeID)
    allocation.Set("mentee_ids", menteeIDs)

    // Debug: Log menteeIDs after appending
    log.Printf("After appending, mentee_ids: %v", menteeIDs)

    // Save the updated allocation record
    err = app.Dao().SaveRecord(allocation)
    if err != nil {
        return fmt.Errorf("failed to update allocation: %w", err)
    }

    log.Println("Successfully updated allocation with new mentee")
    return nil
}
