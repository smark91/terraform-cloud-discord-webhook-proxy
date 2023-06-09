package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

// version is set at build time using -ldflags "-X main.version=$VERSION"
var version string

func main() {
	// Read the Discord webhook URL from an environment variable
	webhookURL := os.Getenv("TF_DISCORD_PROXY_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("TF_DISCORD_PROXY_WEBHOOK_URL environment variable not set")
	}

	// Get auth token for HMAC verification (optional)
	authToken := os.Getenv("TF_DISCORD_PROXY_AUTH_TOKEN")

	// Define the HTTP handler function to handle incoming webhooks
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Write request body to buffer
		body := &bytes.Buffer{}
		_, err := body.ReadFrom(r.Body)
		if err != nil {
			log.Println("Error reading body:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify HMAC signature
		err = verifyHmacSignature(body.Bytes(), r.Header, authToken)
		if err != nil {
			log.Println("Error verifying signature for payload:", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse the incoming JSON payload into a struct
		var payload Payload
		err = json.Unmarshal(body.Bytes(), &payload)
		if err != nil {
			log.Println("Error decoding payload:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create the Discord webhook message from the parsed payload
		message, err := createDiscordMessage(payload)
		if err != nil {
			log.Println("Error creating Discord message:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send the Discord webhook message
		err = sendDiscordWebhook(webhookURL, message, payload)
		if err != nil {
			log.Println("Error sending Discord webhook:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Start the HTTP server
	port := os.Getenv("TF_DISCORD_PROXY_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("terraform-cloud-discord-webhook-proxy version %s\n", version)
	log.Printf("Listening on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Define the struct to parse the incoming JSON payload
type Payload struct {
	PayloadVersion              int            `json:"payload_version"`
	NotificationConfigurationID string         `json:"notification_configuration_id"`
	RunURL                      string         `json:"run_url"`
	RunID                       string         `json:"run_id"`
	RunMessage                  string         `json:"run_message"`
	RunCreatedAt                string         `json:"run_created_at"`
	RunCreatedBy                string         `json:"run_created_by"`
	WorkspaceID                 string         `json:"workspace_id"`
	WorkspaceName               string         `json:"workspace_name"`
	OrganizationName            string         `json:"organization_name"`
	Notifications               []Notification `json:"notifications"`
}

type Notification struct {
	Message      string  `json:"message"`
	Trigger      string  `json:"trigger"`
	RunStatus    string  `json:"run_status"`
	RunUpdatedAt string  `json:"run_updated_at"`
	RunUpdatedBy *string `json:"run_updated_by"`
	RunMessage   string  `json:"run_message"`
}

func verifyHmacSignature(body []byte, headers http.Header, token string) error {
	// auth token feature disabled
	if token == "" {
		return nil
	}

	signature := headers.Get("X-TFE-Notification-Signature")
	if signature == "" {
		return errors.New("request does not have the 'X-TFE-Notification-Signature' header")
	}

	h := hmac.New(sha512.New, []byte(token))
	h.Write(body)

	sha := hex.EncodeToString(h.Sum(nil))

	if signature != sha {
		return errors.New("request signature does not match")
	}

	return nil
}

func createDiscordMessage(payload Payload) (string, error) {
	if len(payload.Notifications) == 0 {
		return "", errors.New("payload does not contain any notifications")
	}

	// Skip the check if trigger is "verification"
	if payload.Notifications[0].Trigger != "verification" {
		if err := validatePayload(payload); err != nil {
			return "", err
		}
	}

	color := getColorForRunStatus(payload.Notifications[0].RunStatus)

	embed := createDiscordEmbed(payload, color)

	message := Message{
		Embeds: []Embed{embed},
	}

	payloadBytes, err := json.Marshal(message)
	if err != nil {
		return "", errors.New("Error creating Discord message")
	}

	return string(payloadBytes), nil
}

func validatePayload(payload Payload) error {
	if payload.Notifications[0].RunStatus == "" {
		return errors.New("payload does not contain a run status")
	}

	if payload.WorkspaceName == "" {
		return errors.New("payload does not contain a workspace name")
	}

	if payload.RunID == "" {
		return errors.New("payload does not contain a run ID")
	}

	if payload.RunURL == "" {
		return errors.New("payload does not contain a run URL")
	}

	if payload.Notifications[0].Message == "" {
		return errors.New("payload does not contain a notification message")
	}

	return nil
}

func getColorForRunStatus(runStatus string) int {
	const (
		ColorGrey   = 0x9e9e9e // Grey
		ColorBlue   = 0x2196f3 // Blue
		ColorYellow = 0xffc107 // Yellow
		ColorGreen  = 0x4caf50 // Green
		ColorRed    = 0xf44336 // Red
		ColorViolet = 0xee82ee // Violet
	)

	switch runStatus {
	case "pending", "discarded", "canceled", "force_canceled":
		return ColorGrey
	case "fetching", "fetching_completed", "pre_plan_running", "pre_plan_completed", "queuing", "plan_queued", "planning", "cost_estimating", "cost_estimated", "policy_checking", "policy_checked", "post_plan_running", "post_plan_completed", "apply_queued", "applying":
		return ColorBlue
	case "policy_override", "planned":
		return ColorYellow
	case "policy_soft_failed", "errored":
		return ColorRed
	case "planned_and_finished", "applied", "confirmed":
		return ColorGreen
	default:
		return ColorViolet
	}
}

func createDiscordEmbed(payload Payload, color int) Embed {
	var fields []Field

	if payload.Notifications[0].Trigger != "verification" {
		fields = []Field{
			{
				Name:   "Workspace",
				Value:  payload.WorkspaceName,
				Inline: true,
			},
			{
				Name:   "Run ID",
				Value:  payload.RunID,
				Inline: true,
			},
			{
				Name:   "Run URL",
				Value:  payload.RunURL,
				Inline: false,
			},
			{
				Name:   "Run Status",
				Value:  payload.Notifications[0].RunStatus,
				Inline: true,
			},
			{
				Name:   "Run Updated At",
				Value:  payload.Notifications[0].RunUpdatedAt,
				Inline: true,
			},
			{
				Name:   "Run Message",
				Value:  payload.RunMessage,
				Inline: false,
			},
		}
	}

	embed := Embed{
		Title:  payload.Notifications[0].Message,
		Color:  color,
		Fields: fields,
	}

	return embed
}

type Message struct {
	Embeds []Embed `json:"embeds"`
}

type Embed struct {
	Title  string  `json:"title"`
	Color  int     `json:"color"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// Define the function to send the Discord webhook message
func sendDiscordWebhook(webhookURL string, message string, payload Payload) error {
	reqBody := []byte(message)

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Println("Error creating HTTP request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending HTTP request:", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var runStatus string

		if payload.Notifications[0].Trigger == "verification" {
			runStatus = "verification"
		} else {
			runStatus = payload.Notifications[0].RunStatus
		}
		log.Println("Discord webhook sent successfully with status:", runStatus)
		return nil
	} else {
		log.Println("Error sending Discord webhook: unexpected status code", resp.StatusCode)
		return errors.New("unexpected status code")
	}
}
