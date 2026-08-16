package notifications

import "testing"

func TestEmailTLSVerificationCannotBeDisabled(t *testing.T) {
	notifier := &emailTypeNotifier{tlsSkipVerify: true}
	if _, err := notifier.GetURL(nil); err == nil {
		t.Fatal("GetURL succeeded with TLS verification disabled")
	}
}
