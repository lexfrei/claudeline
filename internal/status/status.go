// Package status checks the Claude platform status for active incidents.
package status

import (
	"encoding/json"
	"time"

	"github.com/lexfrei/claudeline/internal/cache"
	"github.com/lexfrei/claudeline/internal/httpclient"
)

const (
	cacheTTL   = 15 * time.Second
	apiURL     = "https://status.claude.com/api/v2/status.json"
	apiTimeout = 2 * time.Second
)

// CachePath is the path to the status cache file. Replaceable for testing.
var CachePath = "/tmp/claude-status-cache.json"

// HTTPGetFn is the function used for HTTP requests. Replaceable for testing.
var HTTPGetFn httpclient.GetFn = httpclient.Get

var statusMap = map[string]string{
	"minor":    "⚠️ degraded",
	"major":    "🔶 major outage",
	"critical": "🔴 critical outage",
}

type apiResponse struct {
	Status struct {
		Indicator string `json:"indicator"`
	} `json:"status"`
}

// FetchAlert returns a status string if Claude platform has active incidents.
func FetchAlert() string {
	if cached, ok := cache.Read(CachePath, cacheTTL); ok {
		return string(cached)
	}

	body, err := HTTPGetFn(apiURL, nil, apiTimeout)
	if err != nil {
		cache.Write(CachePath, nil)

		return ""
	}

	var resp apiResponse

	unmarshalErr := json.Unmarshal(body, &resp)
	if unmarshalErr != nil {
		cache.Write(CachePath, nil)

		return ""
	}

	result := statusMap[resp.Status.Indicator]
	cache.Write(CachePath, []byte(result))

	return result
}
