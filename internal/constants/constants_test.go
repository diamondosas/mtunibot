package constants

import (
	"testing"
)

func TestConstantsAndDepartmentMatching(t *testing.T) {
	// Verify departments list
	depts := GetAllDepartments()
	if len(depts) == 0 {
		t.Fatalf("Expected non-empty departments list")
	}

	// Test valid department match
	matchedDept, col, ok := FindMatchingDepartment("computer science & maths")
	if !ok || matchedDept != "Computer Science & Maths" || col != "CBAS" {
		t.Errorf("Expected match for Computer Science & Maths in CBAS, got dept=%q, col=%q, ok=%v", matchedDept, col, ok)
	}

	// Test fuzzy / partial match
	matchedDept2, col2, ok := FindMatchingDepartment("Software Eng")
	if !ok || matchedDept2 != "Software Engineering" || col2 != "CBAS" {
		t.Errorf("Expected match for Software Engineering in CBAS, got dept=%q, col=%q, ok=%v", matchedDept2, col2, ok)
	}

	// Test invalid / NONE department
	_, _, okNone := FindMatchingDepartment("NONE")
	if okNone {
		t.Errorf("Expected 'NONE' to be rejected")
	}

	_, _, okUnknown := FindMatchingDepartment("Mechanical Engineering")
	if okUnknown {
		t.Errorf("Expected unregistered department to be rejected")
	}
}
