package codex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

const (
	maxPowAttempts = 500000
	errorPrefix    = "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D"
)

// fnv1a32 computes the FNV-1a 32-bit hash with xorshift mixing (murmurhash3 finalizer).
// Reverse-engineered from sentinel SDK JS.
func fnv1a32(text string) string {
	h := uint32(2166136261) // FNV offset basis
	for _, ch := range text {
		h ^= uint32(ch)
		h *= 16777619 // FNV prime
	}
	// xorshift mixing (murmurhash3 finalizer)
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return fmt.Sprintf("%08x", h)
}

// base64Encode mimics the SDK's E() function: JSON.stringify → UTF-8 → base64.
func base64Encode(data interface{}) string {
	jsonBytes, _ := json.Marshal(data)
	return base64.StdEncoding.EncodeToString(jsonBytes)
}

// browserConfig builds a fake browser environment array (25 elements).
// Reverse-engineered from sentinel SDK's _getConfig().
func browserConfig(ua, sid string) []interface{} {
	screenVals := []int{2667, 2745, 2880, 3000, 2560, 2200, 2160}
	screenVal := screenVals[rand.Intn(len(screenVals))]

	now := time.Now()
	zone, _ := now.Zone()
	dateStr := now.Format("Mon Jan 02 2006 15:04:05 ") + zone

	scriptSrcs := []string{
		"https://sentinel.openai.com/sentinel/20260219f9f6/sdk.js",
		"https://sentinel.openai.com/backend-api/sentinel/sdk.js",
	}
	builds := []string{"en-US", "zh-CN", "en"}

	navProps := []string{
		"windowControlsOverlay\u2212[object WindowControlsOverlay]",
		"scheduling\u2212[object Scheduling]",
		"pdfViewerEnabled\u2212true",
		"hardwareConcurrency\u221216",
		"deviceMemory\u22128",
		"maxTouchPoints\u22120",
		"cookieEnabled\u2212true",
		"vendor\u2212Google Inc.",
		"language\u2212en-US",
		"onLine\u2212true",
		"webdriver\u2212false",
	}
	docKeys := []string{"location", "implementation", "URL", "documentURI", "compatMode"}
	winKeys := []string{
		"__oai_so_bm", "__oai_logHTML", "__NEXT_DATA__",
		"__next_f", "__oai_SSR_TTI", "__oai_SSR_HTML",
		"__reactEvents", "__RUNTIME_CONFIG__",
	}
	hwConcurrencies := []int{4, 8, 12, 16}

	perfNow := rand.Float64()*49000 + 1000
	hwConc := hwConcurrencies[rand.Intn(len(hwConcurrencies))]
	timeOrigin := float64(time.Now().UnixMilli()) - perfNow

	return []interface{}{
		screenVal,                               // [0]  screen dimensions
		dateStr,                                 // [1]  date string
		4294967296,                              // [2]  jsHeapSizeLimit
		rand.Float64(),                          // [3]  placeholder → nonce
		ua,                                      // [4]  userAgent
		scriptSrcs[rand.Intn(len(scriptSrcs))],  // [5]  script src
		nil,                                     // [6]  script version
		builds[rand.Intn(len(builds))],          // [7]  data-build
		"en-US",                                 // [8]  language
		"en-US,en",                              // [9]  placeholder → elapsed ms
		rand.Float64(),                          // [10] random
		navProps[rand.Intn(len(navProps))],       // [11] navigator property
		docKeys[rand.Intn(len(docKeys))],         // [12] document key
		winKeys[rand.Intn(len(winKeys))],         // [13] window key
		perfNow,                                 // [14] performance.now
		sid,                                     // [15] session UUID
		"",                                      // [16] URL params
		hwConc,                                  // [17] hardwareConcurrency
		timeOrigin,                              // [18] timeOrigin
		0, 0, 0, 0, 0, 0,                       // [19-24] feature flags
	}
}

// generateRequirementsToken generates a token without server-side PoW challenge.
// Used for the initial sentinel/req request's "p" field.
func generateRequirementsToken(ua, sid string) string {
	config := browserConfig(ua, sid)
	config[3] = 1
	config[9] = rand.Intn(45) + 5 // small delay ms
	data := base64Encode(config)
	return "gAAAAAC" + data // note: prefix C, not B
}

// generatePowToken runs the PoW search and returns a sentinel token string.
func generatePowToken(ua, sid, seed, difficulty string) string {
	startTime := time.Now()
	config := browserConfig(ua, sid)

	for i := 0; i < maxPowAttempts; i++ {
		config[3] = i
		config[9] = int(time.Since(startTime).Milliseconds())
		data := base64Encode(config)
		hashHex := fnv1a32(seed + data)

		if len(difficulty) > 0 && hashHex[:len(difficulty)] <= difficulty {
			return "gAAAAAB" + data + "~S"
		}
	}
	// PoW failed — return error token
	errData := base64Encode(fmt.Sprintf("%v", nil))
	return "gAAAAAB" + errorPrefix + errData
}

// SentinelToken holds the JSON token sent as openai-sentinel-token header.
type SentinelToken struct {
	P    string `json:"p"`
	T    string `json:"t"`
	C    string `json:"c"`
	ID   string `json:"id"`
	Flow string `json:"flow"`
}

// BuildSentinelToken fetches the challenge from sentinel API and computes PoW.
func BuildSentinelToken(client *HTTPClient, deviceID, ua, flow string) (string, error) {
	sid := uuid.New().String()
	pToken := generateRequirementsToken(ua, sid)

	reqBody := map[string]string{
		"p":    pToken,
		"id":   deviceID,
		"flow": flow,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	headers := map[string]string{
		"Content-Type":    "text/plain;charset=UTF-8",
		"Referer":         "https://sentinel.openai.com/backend-api/sentinel/frame.html",
		"User-Agent":      ua,
		"Origin":          "https://sentinel.openai.com",
		"sec-ch-ua":       `"Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"`,
		"sec-ch-ua-mobile": "?0",
		"sec-ch-ua-platform": `"Windows"`,
	}

	resp, err := client.Post(
		"https://sentinel.openai.com/backend-api/sentinel/req",
		headers,
		bodyBytes,
	)
	if err != nil {
		return "", fmt.Errorf("sentinel API error: %w", err)
	}

	var challenge struct {
		Token       string `json:"token"`
		ProofOfWork struct {
			Required   bool   `json:"required"`
			Seed       string `json:"seed"`
			Difficulty string `json:"difficulty"`
		} `json:"proofofwork"`
	}
	if err := json.Unmarshal(resp, &challenge); err != nil {
		return "", fmt.Errorf("sentinel response parse error: %w", err)
	}

	var pValue string
	if challenge.ProofOfWork.Required && challenge.ProofOfWork.Seed != "" {
		pValue = generatePowToken(ua, sid, challenge.ProofOfWork.Seed, challenge.ProofOfWork.Difficulty)
	} else {
		pValue = generateRequirementsToken(ua, sid)
	}

	st := SentinelToken{
		P:    pValue,
		T:    "",
		C:    challenge.Token,
		ID:   deviceID,
		Flow: flow,
	}
	tokenBytes, _ := json.Marshal(st)
	return string(tokenBytes), nil
}
