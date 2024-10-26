package handlers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

type SessionHandler struct {
	app *pocketbase.PocketBase
}

func NewSessionHandler(app *pocketbase.PocketBase) *SessionHandler {
	return &SessionHandler{app: app}
}

func (h *SessionHandler) GetMentorSessions(c echo.Context) error {
	// Get mentorID from the URL parameter
	mentorID := "sjhsz1ttnoeamzi" // This should ideally be extracted from the context

	// Find sessions where the mentor relation matches the given mentorID
	sessions, err := h.app.Dao().FindRecordsByExpr("sessions",
		dbx.HashExp{"mentor_id": mentorID},
	)
	if err != nil {
		log.Printf("Error fetching sessions: %v", err)
		return apis.NewApiError(http.StatusInternalServerError, "Failed to fetch sessions", err)
	}

	// Define response structures
	type StudentInfo struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type SessionInfo struct {
		ID       string        `json:"id"`
		Venue    string        `json:"venue"`
		Datetime string        `json:"datetime"`
		Students []StudentInfo `json:"students"`
	}

	// Prepare response
	var response []SessionInfo

	for _, session := range sessions {
		sessionInfo := SessionInfo{
			ID:       session.Id,
			Venue:    session.GetString("venue"),
			Datetime: session.GetString("datetime"),
		}

		// Step 1: Extract session_students IDs from the session
		studentIDs := session.GetStringSlice("session_students")

		// Step 2: Find allocations where these IDs are listed as record IDs
		var allocations []*models.Record
		if len(studentIDs) > 0 {
			allocations, err = h.app.Dao().FindRecordsByExpr("allocations",
				dbx.NewExp("id IN ({:studentIDs}) AND is_assigned = true", dbx.Params{
					"studentIDs": studentIDs,
				}),
			)
			if err != nil {
				log.Printf("Error fetching allocations for session %s: %v", session.Id, err)
				continue // Skip this session if there's an error
			}
		}

		// Step 3: Collect unique student IDs from allocations
		studentMap := make(map[string]bool)
		for _, allocation := range allocations {
			studentID := allocation.Id // Use allocation.Id as it represents the student ID in this context
			if studentID != "" {
				studentMap[studentID] = true
			}
		}

		// Step 4: Fetch student details from the users table
		var students []StudentInfo
		for studentID := range studentMap {
			student, err := h.app.Dao().FindRecordById("users", studentID)
			if err != nil {
				log.Printf("Error fetching student %s: %v", studentID, err)
				continue // Skip this student if there's an error
			}

			studentInfo := StudentInfo{
				ID:    student.Id,
				Name:  student.GetString("name"),
				Email: student.GetString("email"),
			}
			students = append(students, studentInfo)
		}

		sessionInfo.Students = students
		response = append(response, sessionInfo)
	}

	return c.JSON(http.StatusOK, response)
}
