package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Envelope map[string]interface{}

func WriteJSON(w http.ResponseWriter, status int, data Envelope) {
	js, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		fmt.Printf("error marshaling JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	js = append(js, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(js); err != nil {
		fmt.Printf("error writing JSON response: %v", err)
	}
}
