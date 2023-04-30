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

func TestVerifyHmacSignature_NoToken(t *testing.T) {
	token := ""
	body := []byte("data")
	headers := http.Header{}

	err := verifyHmacSignature(body, headers, token)
	if err != nil {
		t.Errorf("verifyHmacSignature returned an error: %v", err)
	}
}

func TestVerifyHmacSignature_ValidSignature(t *testing.T) {
	token := "my-secret-token"
	body := []byte("data")
	signature := "301a279226be9cef7ba4f266495f48afd83fc0be2ea5fc5602abd1c52cc2fe909c4fb328952897136454968a3aebbc03725f4dadd2d9b205bac6474e8eb4667c"
	headers := http.Header{}

	headers.Set("X-TFE-Notification-Signature", signature)

	err := verifyHmacSignature(body, headers, token)
	if err != nil {
		t.Errorf("verifyHmacSignature returned an error: %v", err)
	}
}

func TestVerifyHmacSignature_InvalidSignature(t *testing.T) {
	token := "my-secret-token"
	body := []byte("data")
	signature := "de688a78bea0d3ef4d48f75974a9ffb5aec5f3959b05bba2e62b30b9152db1777dc73a71e13e1db7ac9eb63322319cff63e23d8dc33c54f4c689a59743091971"
	headers := http.Header{}

	headers.Set("X-TFE-Notification-Signature", signature)

	err := verifyHmacSignature(body, headers, token)
	if err == nil {
		t.Error("verifyHmacSignature did not return an error, but should have")
	}
}

func TestVerifyHmacSignature_NoTokenSkipsSignatureCheck(t *testing.T) {
	token := ""
	body := []byte("data")
	signature := "301a279226be9cef7ba4f266495f48afd83fc0be2ea5fc5602abd1c52cc2fe909c4fb328952897136454968a3aebbc03725f4dadd2d9b205bac6474e8eb4667c"
	headers := http.Header{}

	headers.Set("X-TFE-Notification-Signature", signature)

	err := verifyHmacSignature(body, headers, token)
	if err != nil {
		t.Errorf("verifyHmacSignature returned an error: %v", err)
	}
}

func TestVerifyHmacSignature_MissingSignature(t *testing.T) {
	token := "my-secret-token"
	body := []byte("data")
	headers := http.Header{}

	err := verifyHmacSignature(body, headers, token)
	if err == nil {
		t.Error("verifyHmacSignature did not return an error, but should have")
	} else if err.Error() != "request does not have the 'X-TFE-Notification-Signature' header" {
		t.Errorf("verifyHmacSignature returned the wrong error: %v", err)
	}
}
