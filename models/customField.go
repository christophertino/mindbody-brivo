// Brivo Custom Field Data Model

package models

import (
	"fmt"
	"net/http"

	utils "github.com/christophertino/mindbody-brivo"
)

// CustomFields stores data about custom fields attached to a Brivo user
type CustomFields struct {
	Data  []CustomField `json:"data"`
	Count int           `json:"count"`
}

// CustomField stores data for a single custom field
type CustomField struct {
	ID    int    `json:"id,omitempty"`
	Value string `json:"value"`
}

// GetCustomFieldsForUser retrieves any Brivo custom fields attached to userID
func (customFields *CustomFields) GetCustomFieldsForUser(userID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/custom-fields", userID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = utils.DoRequest(req, customFields); err != nil {
		return err
	}

	return nil
}

// GenerateCustomField will create a CustomField{} based on an ID and Value
func GenerateCustomField(customFieldID int, customFieldValue string) *CustomField {
	customField := CustomField{
		ID:    customFieldID,
		Value: customFieldValue,
	}
	return &customField
}

// GetFieldValue searches through []CustomField by ID and returns Value
func GetFieldValue(fieldID int, customFields []CustomField) (string, error) {
	for _, field := range customFields {
		if field.ID == fieldID {
			return field.Value, nil
		}
	}
	return "", fmt.Errorf("Custom field %d not found", fieldID)
}
