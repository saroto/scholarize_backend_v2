package generator

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateResearchPaperUUID() (string, error) {
	// Generate a uuid
	newId := uuid.New()
	// Convert uuid to string
	newIdString := newId.String()
	// Remove the hyphens
	newIdString = strings.Replace(newIdString, "-", "", -1)

	return newIdString, nil
}
