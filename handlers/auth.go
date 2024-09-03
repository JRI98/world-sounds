package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"world-sounds/models"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

func generatePKCE() (string, string, error) {
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return "", "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	encodedVerifier := base64.RawURLEncoding.EncodeToString(verifier)

	hash := sha256.New()
	hash.Write([]byte(encodedVerifier))
	hashedVerifier := hash.Sum(nil)

	challenge := base64.RawURLEncoding.EncodeToString(hashedVerifier)

	return encodedVerifier, challenge, nil
}

func (h *Handler) AuthSignIn(c echo.Context) error {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE: %w", err)
	}

	redirectURL := fmt.Sprintf("%s/ui/signin?challenge=%s", h.AuthPublicBaseURL, challenge)

	c.SetCookie(&http.Cookie{
		Name:     "edgedb-pkce-verifier",
		Value:    verifier,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) AuthCallback(c echo.Context) error {
	code := c.QueryParam("code")

	cookie, err := c.Cookie("edgedb-pkce-verifier")
	if err != nil {
		return fmt.Errorf("failed to get cookie: %w", err)
	}

	url := fmt.Sprintf("%s/token?code=%s&verifier=%s", h.AuthPrivateBaseURL, code, cookie.Value)

	request, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// TODO: don't skip TLS verification, use http.DefaultClient.Do(request) instead. Fine for now as the request is sent via the private network
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d - `%s`", response.StatusCode, string(responseBytes))
	}

	var responseJSON struct {
		AuthToken string `json:"auth_token"`
	}
	err = json.Unmarshal(responseBytes, &responseJSON)
	if err != nil {
		return err
	}

	err = models.GetTx(h.DB, &responseJSON.AuthToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		user, err := models.UserFetch(ctx, tx)
		if err != nil {
			return err
		}

		if user == nil {
			return models.UserCreate(ctx, tx)
		}

		return nil
	})
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{
		Name:     "edgedb-auth-token",
		Value:    responseJSON.AuthToken,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, http.StatusText(http.StatusOK))
}
