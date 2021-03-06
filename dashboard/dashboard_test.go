package dashboard_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/andrewslotin/michael/dashboard"
	"github.com/andrewslotin/michael/deploy"
	"github.com/andrewslotin/michael/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

/*      Test objects      */
type repoMock struct {
	mock.Mock
}

func (m repoMock) All(key string) []deploy.Deploy {
	return m.Called(key).Get(0).([]deploy.Deploy)
}

func (m repoMock) Since(key string, t time.Time) []deploy.Deploy {
	return m.Called(key, t).Get(0).([]deploy.Deploy)
}

/*          Tests         */
func TestDashboard_OneDeploy(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	d := deploy.New(slack.User{ID: "1", Name: "Test User"}, "Test deploy")
	d.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:28 CEST")
	d.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:38 CEST")

	var repo repoMock
	repo.On("All", "key1").Return([]deploy.Deploy{d})

	mux.Handle("/", dashboard.New(repo))

	response, err := http.Get(baseURL + "/key1")
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	require.NoError(t, err)

	expected := "" +
		"Deploy history\n" +
		"--------------\n" +
		"\n" +
		"* Test User was deploying Test deploy since 04 Aug 16 09:28 CEST until 04 Aug 16 09:38 CEST"

	assert.Equal(t, expected, string(bytes.TrimSpace(body)))

	repo.AssertExpectations(t)
}

func TestDashboard_MultipleDeploys(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	d1 := deploy.New(slack.User{ID: "1", Name: "Test User"}, "First deploy")
	d1.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:28 CEST")
	d1.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:38 CEST")

	d2 := deploy.New(slack.User{ID: "1", Name: "Test User"}, "Second deploy")
	d2.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:39 CEST")
	d2.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:40 CEST")
	d2.Aborted = true

	d3 := deploy.New(slack.User{ID: "1", Name: "Test User"}, "Third deploy")
	d3.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:42 CEST")
	d3.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:43 CEST")
	d3.Aborted = true
	d3.AbortReason = "something went wrong"

	d4 := deploy.New(slack.User{ID: "2", Name: "Another User"}, "Forth deploy")
	d4.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:50 CEST")

	var repo repoMock
	repo.On("All", "key1").Return([]deploy.Deploy{d1, d2, d3, d4})

	mux.Handle("/", dashboard.New(repo))

	response, err := http.Get(baseURL + "/key1")
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	require.NoError(t, err)

	expected := "" +
		"Deploy history\n" +
		"--------------\n" +
		"\n" +
		"* Test User was deploying First deploy since 04 Aug 16 09:28 CEST until 04 Aug 16 09:38 CEST\n" +
		"* Test User was deploying Second deploy since 04 Aug 16 09:39 CEST until 04 Aug 16 09:40 CEST (aborted)\n" +
		"* Test User was deploying Third deploy since 04 Aug 16 09:42 CEST until 04 Aug 16 09:43 CEST (aborted, something went wrong)\n" +
		"* Another User is currently deploying Forth deploy since 04 Aug 16 09:50 CEST"

	assert.Equal(t, expected, string(bytes.TrimSpace(body)))

	repo.AssertExpectations(t)
}

func TestDashboard_NoDeploys(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	var repo repoMock
	repo.On("All", "key1").Return([]deploy.Deploy(nil))

	mux.Handle("/", dashboard.New(repo))

	response, err := http.Get(baseURL + "/key1")
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	require.NoError(t, err)

	expected := "" +
		"Deploy history\n" +
		"--------------\n" +
		"\n" +
		"No deploys in channel so far"

	assert.Equal(t, expected, string(bytes.TrimSpace(body)))

	repo.AssertExpectations(t)
}

func TestDashboard_DeploysSince(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	d := deploy.New(slack.User{ID: "1", Name: "Test User"}, "Test deploy")
	d.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:28 CEST")
	d.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:38 CEST")

	timeSince := d.StartedAt.Add(-5 * time.Minute)

	var repo repoMock
	repo.On("Since", "key1", mock.MatchedBy(timeSince.Equal)).Return([]deploy.Deploy{d})

	mux.Handle("/", dashboard.New(repo))

	reqValues := make(url.Values)
	reqValues.Set("since", timeSince.Format(time.RFC3339))

	response, err := http.Get(baseURL + "/key1?" + reqValues.Encode())
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	require.NoError(t, err)

	expected := "" +
		"Deploy history\n" +
		"--------------\n" +
		"\n" +
		"* Test User was deploying Test deploy since 04 Aug 16 09:28 CEST until 04 Aug 16 09:38 CEST"

	assert.Equal(t, expected, string(bytes.TrimSpace(body)))

	repo.AssertExpectations(t)
}

func TestDashboard_DeploysSince_MalformedTimestamp(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	d := deploy.New(slack.User{ID: "1", Name: "Test User"}, "Test deploy")
	d.StartedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:28 CEST")
	d.FinishedAt, _ = time.Parse(time.RFC822, "04 Aug 16 09:38 CEST")

	timeSince := d.StartedAt.Add(-5 * time.Minute)

	var repo repoMock
	repo.On("Since", "key1", mock.MatchedBy(timeSince.Equal)).Return([]deploy.Deploy{d})

	mux.Handle("/", dashboard.New(repo))

	reqValues := make(url.Values)
	reqValues.Set("since", timeSince.Format(time.RFC822))

	response, err := http.Get(baseURL + "/key1?" + reqValues.Encode())
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", response.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	require.NoError(t, err)

	expected := "Malformed time in `since` parameter"
	assert.Equal(t, expected, string(bytes.TrimSpace(body)))
}

func TestDashboard_MissingChannelID(t *testing.T) {
	baseURL, mux, teardown := setup()
	defer teardown()

	var repo repoMock
	repo.On("All", "key1").Return([]deploy.Deploy(nil))

	mux.Handle("/", dashboard.New(repo))

	response, err := http.Get(baseURL + "/")
	require.NoError(t, err)
	response.Body.Close()

	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestChannelIDFromRequest(t *testing.T) {
	examples := map[string]string{
		"/channel1":                        "channel1",
		"/channel2?key=val":                "channel2",
		"/channel3/hello":                  "channel3",
		"/channel4/hello/world/":           "channel4",
		"/channel5/hello/world/?key=val":   "channel5",
		"/channel6/notchannel.txt":         "channel6",
		"/channel7/notchannel.txt?key=val": "channel7",
		"/channel8.txt":                    "channel8",
		"/":                                "",
		"/?key=val":                        "",
	}

	for path, expectedID := range examples {
		req, err := http.NewRequest("GET", path, nil)
		if !assert.NoError(t, err) {
			continue
		}

		assert.Equal(t, expectedID, dashboard.ChannelIDFromRequest(req))
	}
}

func setup() (url string, mux *http.ServeMux, teardownFn func()) {
	mux = http.NewServeMux()
	srv := httptest.NewServer(mux)

	return srv.URL, mux, srv.Close
}
