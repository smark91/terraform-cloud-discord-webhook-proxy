package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSendDiscordWebhook(t *testing.T) {
	// Set up a test HTTP server to handle the webhook request
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Set the Discord webhook URL to the test server URL
	os.Setenv("TF_DISCORD_PROXY_WEBHOOK_URL", testServer.URL)
	defer os.Unsetenv("TF_DISCORD_PROXY_WEBHOOK_URL")

	// Define a test payload for the webhook
	testPayload := Payload{
		PayloadVersion:              1,
		NotificationConfigurationID: "test-config-id",
		RunURL:                      "https://app.terraform.io/app/test-org/test-ws/runs/run-1234567890",
		RunID:                       "run-1234567890",
		RunMessage:                  "",
		RunCreatedAt:                "2022-05-05T10:00:00Z",
		RunCreatedBy:                "test-user",
		WorkspaceID:                 "test-ws",
		WorkspaceName:               "test-workspace",
		OrganizationName:            "test-org",
		Notifications: []Notification{
			{
				Message:      "Test notification message",
				Trigger:      "manual",
				RunStatus:    "planned_and_finished",
				RunUpdatedAt: "2022-05-05T10:30:00Z",
				RunUpdatedBy: nil,
				RunMessage:   "",
			},
		},
	}

	// Send the test webhook
	err := sendDiscordWebhook(testServer.URL, "test-message", testPayload)
	if err != nil {
		t.Errorf("sendDiscordWebhook returned an error: %v", err)
	}

	// Define a test payload for the verification webhook
	verificationPayload := Payload{
		PayloadVersion:              1,
		NotificationConfigurationID: "test-config-id",
		RunURL:                      "",
		RunID:                       "",
		RunMessage:                  "",
		RunCreatedAt:                "",
		RunCreatedBy:                "",
		WorkspaceID:                 "",
		WorkspaceName:               "",
		OrganizationName:            "",
		Notifications: []Notification{
			{
				Message:      "Test verification message",
				Trigger:      "verification",
				RunStatus:    "",
				RunUpdatedAt: "",
				RunUpdatedBy: nil,
				RunMessage:   "",
			},
		},
	}

	// Send the verification webhook
	err = sendDiscordWebhook(testServer.URL, "test-message", verificationPayload)
	if err != nil {
		t.Errorf("sendDiscordWebhook returned an error: %v", err)
	}
}