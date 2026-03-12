package mapper

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, format string, response Response) {
	switch format {
	case "xml":
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_ = xml.NewEncoder(w).Encode(response)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]Response{
			"subsonic-response": response,
		})
	}
}
