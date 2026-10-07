package db

import (
	"os"
	"testing"
)

func setupTestDB(t *testing.T) {
	_ = os.Remove("db/users_test.db")
	_ = os.Remove("db/notes_test.db")
	_ = os.MkdirAll("db", 0777)

	err := InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
}

func TestUserProfileAndSearch(t *testing.T) {
	setupTestDB(t)

	telegramID := int64(999999)
	err := UpdateUserProfile(telegramID, "CBAS", "Computer Science & Maths", 200)
	if err != nil {
		t.Fatalf("UpdateUserProfile failed: %v", err)
	}

	user, err := GetUser(telegramID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.College != "CBAS" || user.Department != "Computer Science & Maths" || user.Level != 200 {
		t.Errorf("Unexpected user profile: %+v", user)
	}

	testHash := "e4d909c290d0fb1ca068ffaddf22cbd0"

	// Test Note saving with Hash
	note := Note{
		CourseCode: "SEN209",
		Title:      "Software Construction",
		College:    "CBAS",
		Department: "Computer Science & Maths",
		Level:      "200 Level",
		FileID:     "file_test_123",
		FileName:   "SENG 209 Note 1.pptx",
		FileHash:   testHash,
		Tags:       "debugging, errors, isolating, defects",
	}

	err = SaveNote(&note)
	if err != nil {
		t.Fatalf("SaveNote failed: %v", err)
	}

	// Test GetNoteByHash
	foundNote, err := GetNoteByHash(testHash)
	if err != nil || foundNote == nil {
		t.Fatalf("GetNoteByHash failed: %v", err)
	}
	if foundNote.CourseCode != "SEN209" {
		t.Errorf("Expected course SEN209, got %s", foundNote.CourseCode)
	}

	// Test GetCoursesByDeptAndLevel
	courses, err := GetCoursesByDeptAndLevel("Computer Science & Maths", "200")
	if err != nil || len(courses) == 0 {
		t.Fatalf("GetCoursesByDeptAndLevel failed: %v (found %d)", err, len(courses))
	}
	if courses[0] != "SEN209" {
		t.Errorf("Expected SEN209, got %s", courses[0])
	}

	// Test GetNotesByCourse
	courseNotes, err := GetNotesByCourse("Computer Science & Maths", "200", "SEN209")
	if err != nil || len(courseNotes) == 0 {
		t.Fatalf("GetNotesByCourse failed: %v", err)
	}

	// Search by course code
	results, err := SearchNotes("SEN209")
	if err != nil || len(results) == 0 {
		t.Fatalf("SearchNotes by course code failed: %v (found %d)", err, len(results))
	}

	// Search by tag
	resultsTag, err := SearchNotes("debugging")
	if err != nil || len(resultsTag) == 0 {
		t.Fatalf("SearchNotes by tag failed: %v (found %d)", err, len(resultsTag))
	}
}
