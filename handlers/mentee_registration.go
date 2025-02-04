package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// MaxMenteesPerMentor is the maximum number of mentees a mentor can have
const MaxMenteesPerMentor = 15

const FixedSessionTime = "14:00" // 2:00 PM in HH:mm format

// HandleMenteeRegistration allocates a mentor to a newly registered mentee
func HandleMenteeRegistration(app *pocketbase.PocketBase) func(e *core.RecordCreateEvent) error {
	return func(e *core.RecordCreateEvent) error {
		// Ensure the record is for a mentee
		if e.Record.Collection().Name != "users" || e.Record.GetString("role") != "mentee" {
			return nil // Not a mentee registration
		}

		menteeID := e.Record.Id
		log.Printf("Registering new mentee with ID: %s", menteeID)

		// Generate or fetch the session_id for the mentor
		sessionID := "session_" + time.Now().Format("20060102_150405") // Example: session_20241206_123400

		// Find an available mentor
		mentor, err := FindAvailableMentor(app)
		if err != nil {
			log.Printf("Error finding available mentor: %v", err)
			return err
		}

		log.Printf("Found mentor %s for mentee %s", mentor.Id, menteeID)

		// Allocate the mentee to the mentor in the given session
		err = AddMenteeToAllocation(app, mentor.Id, menteeID, sessionID)
		if err != nil {
			log.Printf("Error adding mentee to mentor allocation: %v", err)
			return err
		}

		log.Printf("Successfully allocated mentor %s to mentee %s in session %s", mentor.Id, menteeID, sessionID)
		return nil
	}
}

// FindAvailableMentor finds the first mentor with less than MaxMenteesPerMentor mentees
func FindAvailableMentor(app *pocketbase.PocketBase) (*models.Record, error) {
	// Query users collection for mentors
	mentors, err := app.Dao().FindRecordsByExpr("users", dbx.NewExp("role = 'mentor'"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mentors: %w", err)
	}

	if len(mentors) == 0 {
		return nil, fmt.Errorf("no mentors found")
	}

	// Check for mentors with available capacity
	for _, mentor := range mentors {
		// Get the mentee count for the mentor
		menteeCount, err := GetMenteeCountForMentor(app, mentor.Id)
		if err != nil {
			log.Printf("Error getting mentee count for mentor %s: %v", mentor.Id, err)
			continue
		}

		// If mentor has available capacity, return mentor
		if menteeCount < MaxMenteesPerMentor {
			return mentor, nil
		}
	}

	return nil, fmt.Errorf("no mentors with available capacity")
}

// GetMenteeCountForMentor returns the count of mentees assigned to a mentor
func GetMenteeCountForMentor(app *pocketbase.PocketBase, mentorID string) (int, error) {
	// Query allocations collection for mentees assigned to the mentor
	allocations, err := app.Dao().FindRecordsByExpr("allocations", dbx.NewExp("mentor_id = {:mentorID}", dbx.Params{"mentorID": mentorID}))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch allocations for mentor %s: %w", mentorID, err)
	}

	// Count total mentees in all allocations (by session)
	menteeCount := 0
	for _, allocation := range allocations {
		menteeIDs := allocation.GetStringSlice("mentee_ids")
		menteeCount += len(menteeIDs)
	}

	return menteeCount, nil
}

// AddMenteeToAllocation adds a mentee to a mentor's allocation in a specific session
func AddMenteeToAllocation(app *pocketbase.PocketBase, mentorID string, menteeID string, sessionID string) error {
	// Query allocations collection to find mentor's allocation for the given session_id
	allocations, err := app.Dao().FindRecordsByExpr("allocations", dbx.NewExp("mentor_id = {:mentorID} AND session_id = {:sessionID}", dbx.Params{"mentorID": mentorID, "sessionID": sessionID}))
	if err != nil {
		return fmt.Errorf("failed to fetch allocation for mentor %s in session %s: %w", mentorID, sessionID, err)
	}

	var allocation *models.Record
	if len(allocations) > 0 {
		// Use existing allocation if found
		allocation = allocations[0]
	} else {
		// Create a new allocation record if not found
		collection, err := app.Dao().FindCollectionByNameOrId("allocations")
		if err != nil {
			return fmt.Errorf("failed to find allocations collection: %w", err)
		}

		allocation = models.NewRecord(collection)
		allocation.Set("mentor_id", mentorID)
		allocation.Set("session_id", sessionID) // Assign the session_id
		allocation.Set("allocated_on", time.Now().UTC().Format(time.RFC3339))
		allocation.Set("status", "upcoming")
		allocation.Set("is_assigned", true)

		// Set session date and time combined as a single DateTime value
		nextWednesday := getNextWednesday(time.Now())
		sessionDateTime := time.Date(
			nextWednesday.Year(),
			nextWednesday.Month(),
			nextWednesday.Day(),
			14, // Hour: 14 (2 PM)
			0,  // Minute: 00
			0,  // Second: 00
			0,  // Nanosecond: 0
			time.UTC, // Set to UTC timezone
		)

		// Format as ISO 8601 string
		allocation.Set("session_date", sessionDateTime.Format(time.RFC3339))

		// Randomly assign a venue
		venueID, err := GetRandomVenueID(app)
		if err != nil {
			return fmt.Errorf("failed to assign venue: %w", err)
		}
		allocation.Set("venue_id", venueID)
	}

	// Retrieve the current list of mentee IDs (relation field)
	menteeIDs := allocation.GetStringSlice("mentee_ids")

	// Check if the mentee ID is already in the allocation
	for _, id := range menteeIDs {
		if id == menteeID {
			return fmt.Errorf("mentee %s is already assigned to mentor %s in session %s", menteeID, mentorID, sessionID)
		}
	}

	// Check if the mentor has reached the maximum number of mentees
	if len(menteeIDs) >= MaxMenteesPerMentor {
		return fmt.Errorf("mentor %s has reached the maximum mentee limit of %d in session %s", mentorID, MaxMenteesPerMentor, sessionID)
	}

	// Add the new mentee ID to the list of mentee IDs
	menteeIDs = append(menteeIDs, menteeID)

	// Update the mentee_ids relation field with the new list of IDs
	allocation.Set("mentee_ids", menteeIDs)

	// Save the updated allocation record
	if err := app.Dao().SaveRecord(allocation); err != nil {
		return fmt.Errorf("failed to save allocation: %w", err)
	}

	log.Printf("Mentee %s successfully added to mentor %s's allocation in session %s", menteeID, mentorID, sessionID)
	return nil
}


// GetRandomVenueID selects a random venue ID from the venues collection
func GetRandomVenueID(app *pocketbase.PocketBase) (string, error) {
	// Query all venues from the venues collection
	venues, err := app.Dao().FindRecordsByExpr("venues", dbx.NewExp("1 = 1")) // Fetch all venues
	if err != nil {
		return "", fmt.Errorf("failed to fetch venues: %w", err)
	}

	if len(venues) == 0 {
		return "", fmt.Errorf("no venues available")
	}

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Select a random venue
	randomIndex := rand.Intn(len(venues))
	randomVenue := venues[randomIndex]

	return randomVenue.Id, nil
}

// getNextWednesday calculates the next Wednesday from a given date
func getNextWednesday(from time.Time) time.Time {
	targetDay := time.Wednesday
	daysUntilNext := (int(targetDay) - int(from.Weekday()) + 7) % 7
	if daysUntilNext == 0 {
		daysUntilNext = 7 // Move to the next week's Wednesday
	}
	return from.AddDate(0, 0, daysUntilNext)
}
