// Package status checks the Claude platform status for active incidents.
package status

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lexfrei/claudeline/internal/cache"
	"github.com/lexfrei/claudeline/internal/httpclient"
)

const (
	apiURL     = "https://status.claude.com/api/v2/status.json"
	apiTimeout = 2 * time.Second
)

// CacheTTL is the cache duration for status data. Configurable at startup.
var CacheTTL = 15 * time.Second

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
	if cached, ok := cache.Read(CachePath, CacheTTL); ok {
		return string(cached)
	}

	httpResp, err := HTTPGetFn(apiURL, nil, apiTimeout)
	if err != nil {
		_ = cache.Write(CachePath, nil)

		return ""
	}

	if httpResp.StatusCode != http.StatusOK {
		_ = cache.Write(CachePath, nil)

		return ""
	}

	var resp apiResponse

	unmarshalErr := json.Unmarshal(httpResp.Body, &resp)
	if unmarshalErr != nil {
		_ = cache.Write(CachePath, nil)

		return ""
	}

	result := statusMap[resp.Status.Indicator]
	_ = cache.Write(CachePath, []byte(result))

	return result
}
