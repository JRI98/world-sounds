package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"world-sounds/models"
	"world-sounds/services"

	"github.com/edgedb/edgedb-go"
	"github.com/labstack/echo/v4"
)

func (h *Handler) UserFetch(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	var user *models.UserFetchResult
	err = models.GetTx(h.DB, authToken)(context.Background(), func(ctx context.Context, tx *edgedb.Tx) error {
		user, err = models.UserFetch(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	return c.JSON(http.StatusOK, user)
}

type UserUpdateData struct {
	Username *string `json:"username"`
}

func (h *Handler) UserUpdate(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	data, err := validateData[UserUpdateData](c)
	if err != nil {
		return err
	}

	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		err = models.UserUpdate(ctx, tx, data.Username)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UserUpdateImage(c echo.Context) error {
	authToken, err := GetAuthToken(c)
	if err != nil {
		return err
	}

	uploadedImage, err := c.FormFile("image")
	if err != nil {
		return newEchoHTTPError(http.StatusBadRequest, "image not provided", err)
	}
	src, err := uploadedImage.Open()
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer src.Close()

	filePath, err := services.ProcessImage(c.Request().Context(), src)
	if err != nil {
		return fmt.Errorf("failed to process image: %w", err)
	}
	defer os.Remove(filePath)

	imageHash, err := SHA256File(filePath)
	if err != nil {
		return fmt.Errorf("failed to hash image file: %w", err)
	}

	fileLocation, err := h.S3.UploadWebP(c.Request().Context(), filePath, imageHash+".webp")
	if err != nil {
		return fmt.Errorf("failed to upload WebP: %w", err)
	}

	err = models.GetTx(h.DB, authToken)(c.Request().Context(), func(ctx context.Context, tx *edgedb.Tx) error {
		err = models.UserUpdateImage(ctx, tx, fileLocation)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}

	return c.JSON(http.StatusCreated, map[string]any{"image_uri": fileLocation})
}
