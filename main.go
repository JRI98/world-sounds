package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
	"world-sounds/handlers"
	"world-sounds/models"

	"github.com/edgedb/edgedb-go"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "embed"
)

//go:embed index.html
var indexHTML string

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	handler, err := handlers.NewHandler()
	if err != nil {
		slog.Error("Failed to initialize handler", slog.Any("err", err))
		os.Exit(1)
	}
	defer handler.Cleanup()

	e := echo.New()
	e.HideBanner = true
	e.Validator = &CustomValidator{validator: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		httpError, ok := err.(*echo.HTTPError)
		if !ok {
			httpError = echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)).SetInternal(err)
		}

		var sendError error
		if c.Request().Method == http.MethodHead {
			sendError = c.NoContent(httpError.Code)
		} else {
			sendError = c.String(httpError.Code, fmt.Sprint(httpError.Message))
		}

		if sendError != nil {
			slog.Error("HTTPErrorHandler send error", slog.Any("sendError", sendError), slog.Any("httpError", httpError))
		}
	}

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log := slog.Info
			if v.Error != nil {
				log = slog.Error
			}

			log(fmt.Sprintf("%v %v | %v | %v", v.Method, v.URI, v.Status, v.Latency),
				slog.Time("start_time", v.StartTime),
				slog.Duration("latency", v.Latency),
				slog.String("protocol", v.Protocol),
				slog.String("remote_ip", v.RemoteIP),
				slog.String("host", v.Host),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.String("request_id", v.RequestID),
				slog.String("referer", v.Referer),
				slog.String("user_agent", v.UserAgent),
				slog.Int("status", v.Status),
				slog.Any("error", v.Error),
				slog.String("content_length", v.ContentLength),
				slog.Int64("response_size", v.ResponseSize),
			)

			return nil
		},

		LogLatency:       true,
		LogProtocol:      true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogRequestID:     true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogError:         true,
		LogContentLength: true,
		LogResponseSize:  true,
	}))

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			return fmt.Errorf("[PANIC RECOVER] %v\n%s", err, stack)
		},
		DisableErrorHandler: true,
	}))

	e.Use(middleware.RequestID())

	e.Use(middleware.Secure())

	e.Use(middleware.CORS())

	e.Use(middleware.BodyLimit("60MB"))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("edgedb-auth-token")
			if err != nil {
				return next(c)
			}

			c.Set("authToken", cookie.Value)

			// TODO: add user identity to the context for logging

			return next(c)
		}
	})

	e.GET("", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexHTML)
	})

	api := e.Group("/api")

	v1 := api.Group("/v1")

	auth := v1.Group("/auth")
	auth.GET("/signin", handler.AuthSignIn)
	auth.GET("/callback", handler.AuthCallback)

	me := v1.Group("/me")
	me.GET("", handler.UserFetch)
	me.PATCH("", handler.UserUpdate)
	me.PATCH("/image", handler.UserUpdateImage)
	me.GET("/deposits", handler.DepositsFetch)
	me.GET("/bids", handler.BidsFetch)
	me.GET("/stream", handler.StreamFetch)

	deposits := v1.Group("/deposits")
	deposits.POST("/webhook", handler.DepositsWebhook)

	bids := v1.Group("/bids")
	bids.GET("/top", handler.BidsTopFetch)
	bids.POST("", handler.BidsCreate)
	bids.DELETE("/:id", handler.BidsDelete)

	stream := v1.Group("/stream")
	stream.GET("/latest", handler.StreamLatestFetch)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}

		if err := e.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server start error", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	shutdownChannel := make(chan struct{})
	shutdownWaitGroup := &sync.WaitGroup{}

	shutdownWaitGroup.Add(1)
	go func() {
		// TODO: don't run this goroutine in each of the replicas as it is redundant

		defer shutdownWaitGroup.Done()

		var timeout time.Duration
	loop:
		for {
			select {
			case <-time.After(timeout):
				var streamID string
				err = models.GetTx(handler.DB, nil)(context.Background(), func(ctx context.Context, tx *edgedb.Tx) error {
					stream, err := models.StreamLastestFetch(ctx, tx, 0, 1)
					if err != nil {
						return err
					}

					if len(stream) == 1 {
						latestStream := stream[0]
						latestStreamCreatedAt := latestStream.CreatedAt
						latestStreamAudioDuration := time.Duration(latestStream.AudioDurationSeconds) * time.Second
						latestStreamEndTime := latestStreamCreatedAt.Add(latestStreamAudioDuration)
						if time.Until(latestStreamEndTime) > 5*time.Second {
							// Skip as there are more than 5 seconds left on the current audio
							return nil
						}
					}

					bid, err := models.BidsTopDequeue(ctx, tx)
					if err != nil {
						return err
					}

					if bid == nil {
						// Skip as there are no bids
						return nil
					}

					streamID, err = models.StreamCreate(ctx, tx, bid.AudioURI, bid.AudioDurationSeconds, bid.Credits, bid.UserID)
					if err != nil {
						return err
					}

					return nil
				})
				if err != nil {
					timeout = 1 * time.Second
					slog.Error("Failed to insert top bid into stream", slog.Any("err", err))
					continue
				}

				if streamID == "" {
					// TODO: calculate this value based on the time left for the latest stream if there is one. Schedule for 5 seconds before it ends
					timeout = 5 * time.Second
					continue
				}

				slog.Info("Inserted top bid into stream", slog.String("streamID", streamID))

			case <-shutdownChannel:
				break loop
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	close(shutdownChannel)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = e.Shutdown(ctx)
	if err != nil {
		slog.Error("HTTP shutdown error", slog.Any("err", err))
	}

	shutdownWaitGroup.Wait()

	slog.Info("Server successfully shutdown")
}
