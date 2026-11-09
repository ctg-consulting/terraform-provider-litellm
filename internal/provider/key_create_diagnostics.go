package provider

import "fmt"

func generatedKeyCreateError(err error) string {
	classification := ClassifyHTTPFailure(err)
	status := "unavailable"
	if classification.StatusCode != 0 {
		status = fmt.Sprintf("%d", classification.StatusCode)
	}
	return fmt.Sprintf("Unable to create key. HTTP status: %s; request dispatch confirmed: %t; response acceptance confirmed: %t. Response and transport details were omitted because they may contain key material.", status, classification.RequestDispatched, classification.ResponseAccepted)
}
