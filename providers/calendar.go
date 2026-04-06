package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/types"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type calendarEventItem struct {
	Summary        string            `json:"summary"`
	Start          eventTime         `json:"start"`
	End            eventTime         `json:"end"`
	Location       string            `json:"location"`
	HTMLURL        string            `json:"htmlLink"`
	Attendees      []eventAttendee   `json:"attendees"`
	ConferenceData *conferenceData   `json:"conferenceData,omitempty"`
	HangoutLink    string            `json:"hangoutLink"`
}

type conferenceData struct {
	EntryPoints []entryPoint `json:"entryPoints"`
}

type entryPoint struct {
	EntryPointType string `json:"entryPointType"`
	URI            string `json:"uri"`
}

type eventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

type eventAttendee struct {
	Self           bool   `json:"self"`
	ResponseStatus string `json:"responseStatus"`
}

func (e calendarEventItem) toEvent() types.Event {
	start, _ := time.Parse(time.RFC3339, e.Start.DateTime)
	if start.IsZero() {
		start, _ = time.Parse("2006-01-02", e.Start.Date)
	}
	end, _ := time.Parse(time.RFC3339, e.End.DateTime)
	if end.IsZero() {
		end, _ = time.Parse("2006-01-02", e.End.Date)
	}

	status := types.EventTentative
	for _, a := range e.Attendees {
		if a.Self {
			status = mapEventStatus(a.ResponseStatus)
			break
		}
	}

	meetingURL := e.HangoutLink
	if e.ConferenceData != nil {
		for _, ep := range e.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" && ep.URI != "" {
				meetingURL = ep.URI
				break
			}
		}
	}

	return types.Event{
		Title:      e.Summary,
		StartTime:  start,
		EndTime:    end,
		Status:     status,
		Location:   e.Location,
		URL:        e.HTMLURL,
		MeetingURL: meetingURL,
	}
}

func mapEventStatus(s string) types.EventStatus {
	switch s {
	case "accepted":
		return types.EventAccepted
	case "declined":
		return types.EventDeclined
	case "tentative", "needsAction", "":
		return types.EventTentative
	default:
		return types.EventTentative
	}
}

type calendarListResponse struct {
	Items []calendarEventItem `json:"items"`
}

type CalendarProvider struct {
	calendarIDs  []string
	daysAhead    int
	showDeclined bool
	tokenPath    string
	oauthConfig  *oauth2.Config
	client       *http.Client
}

func NewCalendarProvider(cfg config.CalendarConfig, configDir string) (*CalendarProvider, error) {
	credPath := filepath.Join(configDir, "google-credentials.json")
	credBytes, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w (run OAuth setup first)", credPath, err)
	}

	oauthCfg, err := google.ConfigFromJSON(credBytes, "https://www.googleapis.com/auth/calendar.readonly")
	if err != nil {
		return nil, fmt.Errorf("parsing OAuth config: %w", err)
	}

	tokenPath := filepath.Join(configDir, "google-token.json")

	p := &CalendarProvider{
		calendarIDs:  cfg.CalendarIDs,
		daysAhead:    cfg.DaysAhead,
		showDeclined: cfg.ShowDeclined,
		tokenPath:    tokenPath,
		oauthConfig:  oauthCfg,
	}

	client, err := p.getClient()
	if err != nil {
		return nil, fmt.Errorf("setting up OAuth client: %w", err)
	}
	p.client = client

	return p, nil
}

func (p *CalendarProvider) getClient() (*http.Client, error) {
	tok, err := p.loadToken()
	if err != nil {
		return nil, fmt.Errorf("no saved token (run `devdash auth google` first): %w", err)
	}
	return p.oauthConfig.Client(context.Background(), tok), nil
}

func (p *CalendarProvider) loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(p.tokenPath)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	return &tok, nil
}

func (p *CalendarProvider) saveToken(tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling token: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p.tokenPath), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(p.tokenPath, data, 0600)
}

// AuthGoogle runs the interactive OAuth flow: opens a browser, starts a local
// callback server, exchanges the code for a token, and saves it to disk.
func AuthGoogle(configDir string) error {
	credPath := filepath.Join(configDir, "google-credentials.json")
	credBytes, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w\nDownload OAuth credentials from https://console.cloud.google.com/apis/credentials", credPath, err)
	}

	oauthCfg, err := google.ConfigFromJSON(credBytes, "https://www.googleapis.com/auth/calendar.readonly")
	if err != nil {
		return fmt.Errorf("parsing OAuth config: %w", err)
	}

	// Use a local redirect for the OAuth flow.
	oauthCfg.RedirectURL = "http://localhost:19876/callback"

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprintf(w, "Error: no authorization code received.")
			return
		}
		codeCh <- code
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{Addr: ":19876", Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	authURL := oauthCfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Opening browser for Google Calendar authorization...")
	fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n\n", authURL)
	openBrowser(authURL)

	fmt.Println("Waiting for authorization...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		server.Close()
		return err
	}

	server.Close()

	tok, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("exchanging code for token: %w", err)
	}

	tokenPath := filepath.Join(configDir, "google-token.json")
	p := &CalendarProvider{tokenPath: tokenPath}
	if err := p.saveToken(tok); err != nil {
		return err
	}

	fmt.Printf("Token saved to %s\n", tokenPath)
	fmt.Println("Google Calendar is now configured. Run devdash to see your events.")
	return nil
}

func openBrowser(url string) {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open", url}
	case "linux":
		args = []string{"xdg-open", url}
	default:
		return
	}
	cmd := exec.Command(args[0], args[1:]...)
	_ = cmd.Start()
}

func (p *CalendarProvider) Fetch(ctx context.Context) ([]types.Event, error) {
	now := time.Now()
	end := now.AddDate(0, 0, p.daysAhead)

	var allEvents []types.Event

	for _, calID := range p.calendarIDs {
		events, err := p.fetchCalendar(ctx, calID, now, end)
		if err != nil {
			return nil, fmt.Errorf("fetching calendar %s: %w", calID, err)
		}
		allEvents = append(allEvents, events...)
	}

	// Sort by start time.
	for i := 0; i < len(allEvents); i++ {
		for j := i + 1; j < len(allEvents); j++ {
			if allEvents[j].StartTime.Before(allEvents[i].StartTime) {
				allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
			}
		}
	}

	if !p.showDeclined {
		var filtered []types.Event
		for _, e := range allEvents {
			if e.Status != types.EventDeclined {
				filtered = append(filtered, e)
			}
		}
		allEvents = filtered
	}

	return allEvents, nil
}

func (p *CalendarProvider) fetchCalendar(ctx context.Context, calID string, start, end time.Time) ([]types.Event, error) {
	u := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?%s",
		url.PathEscape(calID),
		url.Values{
			"timeMin":      {start.Format(time.RFC3339)},
			"timeMax":      {end.Format(time.RFC3339)},
			"singleEvents": {"true"},
			"orderBy":      {"startTime"},
			"maxResults":   {"50"},
		}.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Calendar API returned %d", resp.StatusCode)
	}

	var listResp calendarListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var events []types.Event
	for _, item := range listResp.Items {
		events = append(events, item.toEvent())
	}

	return events, nil
}
