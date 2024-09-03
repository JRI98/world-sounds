package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"world-sounds/models"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

type DepositsWebhookData struct {
	EventType string `json:"event_type" validate:"required"`
	Data      struct {
		TransactionID string `json:"id" validate:"required"`
		Items         []struct {
			Price struct {
				ID         string `json:"id" validate:"required"`
				CustomData struct {
					Seconds string `json:"seconds" validate:"required"`
				} `json:"custom_data" validate:"required"`
			} `json:"price" validate:"required"`
			Quantity int64 `json:"quantity" validate:"required"`
		} `json:"items" validate:"required"`
		CustomData struct {
			UserID string `json:"user_id" validate:"required"`
		} `json:"custom_data" validate:"required"`
	} `json:"data" validate:"required"`
}

func (h *Handler) DepositsWebhook(c echo.Context) error {
	ok, err := h.Paddle.WebhookVerifier.Verify(c.Request())
	if err != nil || !ok {
		return newEchoHTTPError(http.StatusBadRequest, "Paddle webhook verification failed", err)
	}

	webhookInfo, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	c.Request().Body = io.NopCloser(bytes.NewBuffer(webhookInfo))

	data, err := validateData[DepositsWebhookData](c)
	if err != nil {
		return err
	}

	if data.EventType != "transaction.completed" {
		return newEchoHTTPError(http.StatusBadRequest, "event_type must be 'transaction.completed'", nil)
	}

	userID, err := edgedb.ParseUUID(data.Data.CustomData.UserID)
	if err != nil {
		return newEchoHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid user_id: %s", data.Data.CustomData.UserID), err)
	}

	depositedCredits := int64(0)
	for _, item := range data.Data.Items {
		secondsString := item.Price.CustomData.Seconds
		seconds, err := strconv.ParseInt(secondsString, 10, 64)
		if err != nil {
			return newEchoHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid seconds: %s", secondsString), err)
		}
		depositedCredits += item.Quantity * seconds
	}

	var depositID string
	err = models.GetTx(h.DB, nil)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		depositID, err = models.DepositCreate(ctx, tx, depositedCredits, webhookInfo, data.Data.TransactionID, userID)
		if err != nil {
			return err
		}

		err = models.UserIncrementCredits(ctx, tx, userID, depositedCredits)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": depositID})
}

func (h *Handler) DepositsFetch(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	deposits := []models.DepositsFetchResult{}
	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		deposits, err = models.DepositsFetch(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, deposits)
}
