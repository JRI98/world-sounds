package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"world-sounds/models"
	"world-sounds/services"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

func (h *Handler) BidsCreate(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	creditsFormValue := c.FormValue("credits")
	creditsData, err := strconv.ParseInt(creditsFormValue, 10, 64)
	if err != nil {
		return newEchoHTTPError(http.StatusBadRequest, "Credits must be an integer", err)
	}

	{
		// Check if user has enough credits before handling the upload. This value will again be enforced when creating the bid
		var userCredits int64
		err := models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
			user, err := models.UserFetch(ctx, tx)
			if err != nil {
				return err
			}

			userCredits = user.Credits

			return nil
		})
		if err != nil {
			return err
		}

		if creditsData > userCredits {
			return newEchoHTTPError(http.StatusBadRequest, "Credits must be less than or equal to user credits", nil)
		}
	}

	uploadedAudio, err := c.FormFile("audio")
	if err != nil {
		return err
	}
	src, err := uploadedAudio.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	filePath, durationSeconds, err := services.ProcessAudio(c.Request().Context(), src)
	if err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}
	defer os.Remove(filePath)

	if durationSeconds == 0 {
		return newEchoHTTPError(http.StatusBadRequest, "Duration must not be zero", nil)
	}

	if creditsData < durationSeconds {
		return newEchoHTTPError(http.StatusBadRequest, "Credits must be greater than or equal to duration", nil)
	}

	fileHash, err := SHA256File(filePath)
	if err != nil {
		return fmt.Errorf("failed to hash audio file: %w", err)
	}

	fileLocation, err := h.S3.UploadMP3(c.Request().Context(), filePath, fileHash+".mp3")
	if err != nil {
		return fmt.Errorf("failed to upload MP3: %w", err)
	}

	var bidID string
	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		err = models.UserDecrementCredits(ctx, tx, creditsData)
		if err != nil {
			return err
		}

		bidID, err = models.BidCreate(ctx, tx, fileLocation, durationSeconds, creditsData)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": bidID})
}

func (h *Handler) BidsFetch(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	feed := []models.BidsFetchResult{}
	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		feed, err = models.BidsFetch(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, feed)
}

func (h *Handler) BidsTopFetch(c echo.Context) error {
	var err error
	feed := []models.BidsTopFetchResult{}
	err = models.GetTx(h.DB, nil)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		feed, err = models.BidsTopFetch(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, feed)
}

type BidsDeleteData struct {
	BidID edgedb.UUID `json:"id" validate:"required"`
}

func (h *Handler) BidsDelete(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	data, err := validateData[BidsDeleteData](c)
	if err != nil {
		return err
	}

	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		return models.BidDelete(ctx, tx, data.BidID)
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, http.StatusText(http.StatusOK))
}
