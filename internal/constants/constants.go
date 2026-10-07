package constants

import (
	"fmt"
	"strings"
)

// Colleges list - add new colleges here anytime
var Colleges = []string{
	"CBAS",
	"CHMS",
	"CAHS",
}

// CollegeDepartments map - add or edit departments under their college here
var CollegeDepartments = map[string][]string{
	"CBAS": {
		"Computer Science & Mathematics",
		"Biological Sciences",
		"Biochemistry",
		"Food Science & Technology",
		"Physics",
		"Chemical Sciences",
		"Geosciences",
	},
	"CHMS": {
		"Accounting & Finance",
		"Business Administration",
		"Economics",
		"Fine & Applied Arts",
		"Languages",
		"Mass Communication",
		"Music",
		"Philosophy & Religion Studies",
	},
	"CAHS": {
		"Nursing Science",
		"Public Health",
		"Nutrition & Dietetics",
		"Medical Laboratory Science",
		"Biomedical Technology",
	},
}

// Levels list - add or edit academic levels here
var Levels = []int{100, 200, 300, 400}

// GetAllDepartments returns a flat slice of all registered department names
func GetAllDepartments() []string {
	var list []string
	for _, depts := range CollegeDepartments {
		list = append(list, depts...)
	}
	return list
}

// GetAllColleges returns all registered colleges
func GetAllColleges() []string {
	return Colleges
}

// GetAllLevelsFormatted returns levels formatted e.g. "100 Level", "200 Level"
func GetAllLevelsFormatted() []string {
	var list []string
	for _, l := range Levels {
		list = append(list, fmt.Sprintf("%d Level", l))
	}
	return list
}

// FindMatchingDepartment checks if a given department name matches any registered department (case-insensitive & fuzzy)
func FindMatchingDepartment(deptName string) (matchedDept string, college string, ok bool) {
	cleaned := strings.ToLower(strings.TrimSpace(deptName))
	if cleaned == "" || cleaned == "none" || cleaned == "unknown" {
		return "", "", false
	}

	for col, depts := range CollegeDepartments {
		for _, d := range depts {
			dLower := strings.ToLower(d)
			// Direct match or substring match
			if dLower == cleaned || strings.Contains(cleaned, dLower) || strings.Contains(dLower, cleaned) {
				return d, col, true
			}
		}
	}
	return "", "", false
}
