package slackmessenger

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
)

func TestPostMessage(t *testing.T) {

	testLogger := test.NewGlobal()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer mockServer.Close()
	client := slack.New("YOUR_TEST_TOKEN", slack.OptionAPIURL(mockServer.URL+"/foo"))

	SendSlackMessage(client, "555", "foo bar", Danger)
	assert.Equal(t, 1, len(testLogger.Entries))
	assert.Contains(t, testLogger.Entries[0].Message, "Failed to send slack message")

}
