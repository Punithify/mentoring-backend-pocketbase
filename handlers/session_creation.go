package handlers

import (
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

const studentsPerSession = 5 // Number of students per session

// CreateSessions creates sessions for mentors that have active allocations, grouping allocations into sessions of up to 5 students each.
func CreateSessions(app *pocketbase.PocketBase) error {
	// Step 1: Fetch all mentors (users) with the role "mentor"
	mentors := []*models.Record{}
	err := app.Dao().DB().
		Select("id").
		From("users").
		Where(dbx.HashExp{"role": "mentor"}).
		All(&mentors)
	if err != nil {
		return err
	}

	if len(mentors) == 0 {
		log.Println("No mentors found.")
		return nil
	}

	// Step 2: Fetch the "sessions" collection
	sessionsCollection, err := app.Dao().FindCollectionByNameOrId("sessions")
	if err != nil || sessionsCollection == nil {
		log.Println("Sessions collection not found.")
		return err
	}

	// Step 3: For each mentor, group their allocations into sessions and create session records
	for _, mentor := range mentors {
		mentorID := mentor.Id
		err := createSessionsForMentor(app, mentorID, sessionsCollection)
		if err != nil {
			log.Printf("Error creating sessions for mentor %s: %v\n", mentorID, err)
		}
	}

	log.Println("Session creation for all mentors completed successfully.")
	return nil
}

// createSessionsForMentor creates sessions for a specific mentor, grouping allocations into sessions of up to 5 students each.
func createSessionsForMentor(app *pocketbase.PocketBase, mentorID string, sessionsCollection *models.Collection) error {
	// Fetch all active allocations for the mentor
	allocations := []*models.Record{}
	err := app.Dao().DB().
		Select("*").
		From("allocations").
		Where(dbx.HashExp{"mentor_id": mentorID, "status": "active"}).
		All(&allocations)
	if err != nil {
		return err
	}

	if len(allocations) == 0 {
		log.Printf("No active allocations found for mentor %s.\n", mentorID)
		return nil
	}

	// Group allocations into sessions of up to 5 students each and create session records
	for i := 0; i < len(allocations); i += studentsPerSession {
		end := min(i+studentsPerSession, len(allocations))
		sessionStudents := allocations[i:end]

		// Create a new session record
		session := models.NewRecord(sessionsCollection)
		session.Set("mentor_id", mentorID)
		session.Set("session_students", extractAllocationIDs(sessionStudents))
		session.Set("venue", "Room 101")                        // Example venue
		session.Set("datetime", time.Now().Format(time.RFC3339)) // Example datetime

		err := app.Dao().SaveRecord(session)
		if err != nil {
			log.Println("Failed to create session:", err)
		} else {
			log.Printf("Successfully created a session for mentor %s with %d students\n", mentorID, len(sessionStudents))

			// Mark the allocations as completed
			for _, allocation := range sessionStudents {
				log.Printf("Marking allocation %s as completed", allocation.Id)
				err := markAllocationAsCompleted(app, allocation)
				if err != nil {
					log.Printf("Failed to mark allocation %s as completed: %v", allocation.Id, err)
				}
			}
		}
	}

	return nil
}

// markAllocationAsCompleted reloads the allocation record and updates the status field to "completed"
func markAllocationAsCompleted(app *pocketbase.PocketBase, allocation *models.Record) error {
	// Reload the allocation record to ensure data is loaded
	allocationsCollection, err := app.Dao().FindCollectionByNameOrId("allocations")
	if err != nil {
		log.Printf("Error fetching allocations collection: %v", err)
		return err
	}

	reloadedAllocation, err := app.Dao().FindRecordById(allocationsCollection.Id, allocation.Id)
	if err != nil {
		log.Printf("Error reloading allocation %s: %v", allocation.Id, err)
		return err
	}

	// Set the `status` field to "completed"
	reloadedAllocation.Set("status", "completed")

	// Save the updated allocation record
	err = app.Dao().SaveRecord(reloadedAllocation)
	if err != nil {
		log.Printf("Error saving allocation %s with status=completed: %v", reloadedAllocation.Id, err)
	} else {
		log.Printf("Successfully updated allocation %s to status=completed", reloadedAllocation.Id)
	}
	return err
}

// Helper function to extract allocation IDs
func extractAllocationIDs(allocations []*models.Record) []string {
	allocationIDs := []string{}
	for _, allocation := range allocations {
		if allocation != nil {
			allocationID := allocation.Id
			if allocationID != "" {
				allocationIDs = append(allocationIDs, allocationID)
			} else {
				log.Println("Skipping allocation with missing ID")
			}
		} else {
			log.Println("Encountered nil allocation while extracting IDs")
		}
	}
	return allocationIDs
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
