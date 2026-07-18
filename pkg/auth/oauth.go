package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// defaultHeadless controls whether OAuth flows use manual code paste by default.
var defaultHeadless bool

// SetDefaultHeadless sets the default headless mode for all OAuth flows.
// When true, the OAuth flow uses manual code paste instead of a local HTTP server.
func SetDefaultHeadless(headless bool) {
	defaultHeadless = headless
}

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	CredentialsPath string
	TokenPath       string
	Scopes          []string
	AccountName     string // Optional account name for better user messaging
	Headless        bool   // Use manual code paste instead of local HTTP server
}

// NewOAuth2Config creates a new OAuth2 configuration
func NewOAuth2Config(credentialsPath string, tokenPath string, scopes ...string) *OAuth2Config {
	return &OAuth2Config{
		CredentialsPath: credentialsPath,
		TokenPath:       tokenPath,
		Scopes:          scopes,
		Headless:        defaultHeadless,
	}
}

// SetAccountName sets the account name for better user messaging during OAuth
func (c *OAuth2Config) SetAccountName(accountName string) {
	c.AccountName = accountName
}

// SetHeadless sets headless mode for manual code paste
func (c *OAuth2Config) SetHeadless(headless bool) {
	c.Headless = headless
}

// LoadCredentials loads OAuth2 credentials from file
func (c *OAuth2Config) LoadCredentials() (*oauth2.Config, error) {
	data, err := os.ReadFile(c.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read credentials file: %w", err)
	}

	config, err := google.ConfigFromJSON(data, c.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("could not parse credentials file: %w", err)
	}

	return config, nil
}

// LoadToken loads cached token from file
func (c *OAuth2Config) LoadToken(config *oauth2.Config) (*oauth2.Token, error) {
	f, err := os.Open(c.TokenPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// SaveToken saves token to file
func (c *OAuth2Config) SaveToken(token *oauth2.Token) error {
	// Ensure directory exists
	dir := filepath.Dir(c.TokenPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	f, err := os.OpenFile(c.TokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not save OAuth token: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	// #nosec G117 -- persisting the OAuth token to disk is the intended behaviour; the file is created with 0600 perms above to keep it private to the user.
	return json.NewEncoder(f).Encode(token)
}

// GetToken retrieves a token, refreshing if necessary
func (c *OAuth2Config) GetToken(ctx context.Context) (*oauth2.Token, error) {
	config, err := c.LoadCredentials()
	if err != nil {
		return nil, err
	}

	// Try to load cached token
	token, err := c.LoadToken(config)
	if err != nil {
		// Token not found, need to authenticate
		token, err = c.authenticate(ctx, config)
		if err != nil {
			return nil, err
		}
	}

	// Refresh token if needed
	if !token.Valid() {
		token, err = c.refreshToken(ctx, config, token)
		if err != nil {
			// Check if refresh token is invalid (expired or revoked)
			if strings.Contains(err.Error(), "invalid_grant") ||
				strings.Contains(err.Error(), "Token has been expired or revoked") {
				// Refresh token is invalid, need to re-authenticate
				fmt.Println("\n⚠️  Your Gmail access token has expired or been revoked.")
				fmt.Println("🔐 Re-authentication is required to continue using Gmail TUI.")
				token, err = c.authenticate(ctx, config)
				if err != nil {
					return nil, fmt.Errorf("re-authentication failed: %w", err)
				}
				fmt.Println("✅ Successfully re-authenticated! Gmail TUI is ready to use.")
			} else {
				return nil, fmt.Errorf("token refresh failed: %w", err)
			}
		}
	}

	// Save refreshed token
	if err := c.SaveToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

// authenticate performs OAuth2 authentication
func (c *OAuth2Config) authenticate(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	if c.Headless {
		return c.authenticateManual(ctx, config)
	}
	return c.authenticateWithServer(ctx, config)
}

// authenticateManual performs OAuth2 authentication via manual code paste (for headless/remote use)
func (c *OAuth2Config) authenticateManual(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Use localhost redirect (required by Google Desktop app type)
	localConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  "http://localhost:8080",
		Scopes:       config.Scopes,
		Endpoint:     config.Endpoint,
	}

	authURL := localConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	if c.AccountName != "" {
		fmt.Printf("\n🔐 Authorization required for account: %s\n", c.AccountName)
	} else {
		fmt.Printf("\n🔐 Authorization required\n")
	}
	fmt.Printf("1. Open this link:\n   %s\n\n", authURL)
	fmt.Printf("2. Grant access to the application\n")
	fmt.Printf("3. The browser will redirect to a URL like:\n")
	fmt.Printf("   http://localhost:8080/?code=4/0A...&scope=...\n\n")
	fmt.Printf("4. Copy the authorization code from the URL and paste it below\n\n")

	// Read code from stdin
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Paste authorization code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}
	code = strings.TrimSpace(code)

	// Handle full URL paste — extract just the code parameter
	if strings.Contains(code, "code=") {
		start := strings.Index(code, "code=") + 5
		end := strings.Index(code[start:], "&")
		if end == -1 {
			code = code[start:]
		} else {
			code = code[start : start+end]
		}
		code = strings.TrimSpace(code)
	}

	if code == "" {
		return nil, fmt.Errorf("authorization code was empty")
	}

	// Exchange code for token
	token, err := localConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("could not exchange authorization code for token: %w", err)
	}

	fmt.Printf("✅ Authorization successful!\n")
	return token, nil
}

// authenticateWithServer performs OAuth2 authentication with a local HTTP server
func (c *OAuth2Config) authenticateWithServer(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Create a local server to capture the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start local server
	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code != "" {
				// Send success response
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`
					<html>
						<body>
                            <h2>Authorization successful</h2>
                            <p>You can close this window and return to the application.</p>
						</body>
					</html>
				`))
				codeChan <- code
			} else {
				// Send error response
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`
					<html>
						<body>
                            <h2>Authorization error</h2>
                            <p>Authorization code not received.</p>
						</body>
					</html>
				`))
				errorChan <- fmt.Errorf("authorization code not received")
			}
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	// Create OAuth2 config with local redirect URI
	localConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  "http://localhost:8080",
		Scopes:       config.Scopes,
		Endpoint:     config.Endpoint,
	}

	authURL := localConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if c.AccountName != "" {
		fmt.Printf("\n🔐 Authorization required for account: %s\n", c.AccountName)
	} else {
		fmt.Printf("\n🔐 Authorization required\n")
	}
	fmt.Printf("1. Open this link: %s\n", authURL)
	fmt.Printf("2. Grant access to the application\n")
	fmt.Printf("3. You will be redirected automatically\n")
	if c.AccountName != "" {
		fmt.Printf("\nWaiting for authorization for %s...\n", c.AccountName)
	} else {
		fmt.Printf("\nWaiting for authorization...\n")
	}

	// Wait for authorization code
	var authCode string
	select {
	case authCode = <-codeChan:
		// Success
	case err := <-errorChan:
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("local server error: %w", err)
	case <-time.After(5 * time.Minute):
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("authorization timeout exceeded")
	}

	// Shutdown server
	_ = server.Shutdown(ctx)

	// Exchange code for token
	token, err := localConfig.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("could not exchange authorization code for token: %w", err)
	}

	fmt.Printf("✅ Authorization successful!\n")
	return token, nil
}

// refreshToken refreshes an expired token
func (c *OAuth2Config) refreshToken(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("could not refresh token: %w", err)
	}

	return newToken, nil
}

// NewGmailService creates a new Gmail service using OAuth2
func NewGmailService(ctx context.Context, credentialsPath, tokenPath string, scopes ...string) (*gmail.Service, error) {
	return NewGmailServiceWithAccount(ctx, credentialsPath, tokenPath, "", scopes...)
}

// NewGmailServiceWithAccount creates a new Gmail service using OAuth2 with account context for better user messaging
func NewGmailServiceWithAccount(ctx context.Context, credentialsPath, tokenPath string, accountName string, scopes ...string) (*gmail.Service, error) {
	oauthConfig := NewOAuth2Config(credentialsPath, tokenPath, scopes...)

	// Set account context for better user messaging during OAuth
	oauthConfig.SetAccountName(accountName)

	token, err := oauthConfig.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	config, err := oauthConfig.LoadCredentials()
	if err != nil {
		return nil, err
	}

	httpClient := config.Client(ctx, token)

	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("could not create Gmail service: %w", err)
	}

	return service, nil
}

// NewCalendarService creates a new Google Calendar service using OAuth2
func NewCalendarService(ctx context.Context, credentialsPath, tokenPath string, scopes ...string) (*calapi.Service, error) {
	oauthConfig := NewOAuth2Config(credentialsPath, tokenPath, scopes...)

	token, err := oauthConfig.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	config, err := oauthConfig.LoadCredentials()
	if err != nil {
		return nil, err
	}

	httpClient := config.Client(ctx, token)

	service, err := calapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("could not create Calendar service: %w", err)
	}

	return service, nil
}
