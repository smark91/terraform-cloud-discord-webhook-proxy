package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

func main() {
	// Read the Discord webhook URL from an environment variable
	webhookURL := os.Getenv("TF_DISCORD_PROXY_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("TF_DISCORD_PROXY_WEBHOOK_URL environment variable not set")
	}

	// Define the HTTP handler function to handle incoming webhooks
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Parse the incoming JSON payload into a struct
		var payload Payload
		err := json.NewDecoder(r.Body).Decode(&payload)
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

	// Start the HTTP server
	port := os.Getenv("TF_DISCORD_PROXY_PORT")
	if port == "" {
		port = "8080"
	}
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

func createDiscordMessage(payload Payload) (string, error) {
	if len(payload.Notifications) == 0 {
		return "", errors.New("payload does not contain any notifications")
	}

	// Skip the check if trigger is "verification"
	if payload.Notifications[0].Trigger != "verification" {
		if payload.Notifications[0].RunStatus == "" {
			return "", errors.New("payload does not contain a run status")
		}

		if payload.WorkspaceName == "" {
			return "", errors.New("payload does not contain a workspace name")
		}

		if payload.RunID == "" {
			return "", errors.New("payload does not contain a run ID")
		}

		if payload.RunURL == "" {
			return "", errors.New("payload does not contain a run URL")
		}

		if payload.Notifications[0].Message == "" {
			return "", errors.New("payload does not contain a notification message")
		}
	}

	var color int

	const (
		ColorGrey   = 0x9e9e9e // Grey
		ColorBlue   = 0x2196f3 // Blue
		ColorYellow = 0xffc107 // Yellow
		ColorGreen  = 0x4caf50 // Green
		ColorRed    = 0xf44336 // Red
		ColorViolet = 0xee82ee // Violet
	)

	switch payload.Notifications[0].RunStatus {
	case "pending", "discarded", "canceled", "force_canceled":
		color = ColorGrey
	case "fetching", "fetching_completed", "pre_plan_running", "pre_plan_completed", "queuing", "plan_queued", "planning", "cost_estimating", "cost_estimated", "policy_checking", "policy_checked", "post_plan_running", "post_plan_completed", "apply_queued", "applying":
		color = ColorBlue
	case "policy_override", "planned":
		color = ColorYellow
	case "policy_soft_failed", "errored":
		color = ColorRed
	case "planned_and_finished", "applied", "confirmed":
		color = ColorGreen
	default:
		color = ColorViolet
	}

    var embed Embed

    if payload.Notifications[0].Trigger == "verification" {
        embed = Embed{
            Title:  payload.Notifications[0].Message,
            Color:  color,
            Fields: nil,
        }
    } else {
        embed = Embed{
            Title: payload.Notifications[0].Message,
            Color: color,
            Fields: []Field{
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
            },
        }
    }
    
    message := Message{
        Embeds: []Embed{embed},
    }
    
    payloadBytes, err := json.Marshal(message)
    if err != nil {
        return "", errors.New("Error creating Discord message")
    }
    
    return string(payloadBytes), nil    
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

	return nil
}
