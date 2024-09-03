package handlers

import (
	"context"
	"fmt"
	"net/http"
	"world-sounds/models"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

func (h *Handler) StreamFetch(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	var result []models.StreamFetchResult
	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		result, err = models.StreamFetch(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type StreamFetchData struct {
	Offset int64 `query:"offset"`
	Limit  int64 `query:"limit"`
}

func (h *Handler) StreamLatestFetch(c echo.Context) error {
	data, err := validateData[StreamFetchData](c)
	if err != nil {
		return err
	}

	if data.Offset < 0 {
		return newEchoHTTPError(http.StatusBadRequest, "offset must be greater than or equal to 0", nil)
	}

	if data.Limit == 0 {
		data.Limit = 20
	} else if data.Limit < 1 || data.Limit > 100 {
		return newEchoHTTPError(http.StatusBadRequest, "limit must be between 1 and 100", nil)
	}

	var result []models.StreamLatestFetchResult
	err = models.GetTx(h.DB, nil)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		result, err = models.StreamLastestFetch(ctx, tx, data.Offset, data.Limit)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to fetch latest stream: %w", err)
	}

	return c.JSON(http.StatusOK, result)
}
