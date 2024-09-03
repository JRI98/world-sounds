package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"world-sounds/models"
	"world-sounds/services"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

type (
	Handler struct {
		DB                 *edgedb.Client
		S3                 *services.S3Service
		Paddle             *services.PaddleService
		AuthPublicBaseURL  string
		AuthPrivateBaseURL string
	}
)

func NewHandler() (*Handler, error) {
	dbService, err := models.NewDBService()
	if err != nil {
		return nil, fmt.Errorf("failed to create DB service: %w", err)
	}

	s3Service, err := services.NewS3Service()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 service: %w", err)
	}

	paddleService, err := services.NewPaddleService()
	if err != nil {
		return nil, fmt.Errorf("failed to create Paddle service: %w", err)
	}

	edgedbAuthPublicBaseURL, ok := os.LookupEnv("EDGEDB_AUTH_PUBLIC_BASE_URL")
	if !ok {
		return nil, errors.New("EDGEDB_AUTH_PUBLIC_BASE_URL environment variable not set")
	}

	edgedbAuthPrivateBaseURL, ok := os.LookupEnv("EDGEDB_AUTH_PRIVATE_BASE_URL")
	if !ok {
		return nil, errors.New("EDGEDB_AUTH_PRIVATE_BASE_URL environment variable not set")
	}

	return &Handler{
		DB:                 dbService,
		S3:                 s3Service,
		Paddle:             paddleService,
		AuthPublicBaseURL:  edgedbAuthPublicBaseURL,
		AuthPrivateBaseURL: edgedbAuthPrivateBaseURL,
	}, nil
}

func (h *Handler) Cleanup() {
	err := h.DB.Close()
	if err != nil {
		slog.Error("Error closing DB", slog.Any("err", err))
	}
}

func validateData[T any](c echo.Context) (*T, error) {
	res := new(T)

	if err := c.Bind(res); err != nil {
		return res, err
	}

	if err := c.Validate(res); err != nil {
		return res, err
	}

	return res, nil
}

func GetAuthToken(c echo.Context) (*string, error) {
	authToken, ok := c.Get("authToken").(string)
	if !ok {
		return nil, newEchoHTTPError(http.StatusUnauthorized, "Unauthorized: auth token not provided", nil)
	}
	return &authToken, nil
}

func newEchoHTTPError(code int, message string, err error) *echo.HTTPError {
	return echo.NewHTTPError(code, message).SetInternal(err)
}

func SHA256File(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	sha256Hash := sha256.New()
	if _, err := io.Copy(sha256Hash, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(sha256Hash.Sum(nil)), nil
}
