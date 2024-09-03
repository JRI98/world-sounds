package models

import (
	"context"
	"errors"
	"time"

	"github.com/edgedb/edgedb-go"
)

type BidCreateResult struct {
	ID edgedb.UUID `edgedb:"id"`
}

func BidCreate(ctx context.Context, tx *edgedb.Tx, audioURI string, audioDurationSeconds int64, credits int64) (string, error) {
	var result BidCreateResult

	err := tx.QuerySingle(
		ctx,
		`INSERT Bid {
			audio_uri := <str>$audio_uri,
			audio_duration_seconds := <int64>$audio_duration_seconds,
			credits := <int64>$credits,
			user := (
				select User
				filter .identity = (global ext::auth::ClientTokenIdentity)
			)
		}`,
		&result,
		map[string]interface{}{
			"audio_uri":              audioURI,
			"audio_duration_seconds": audioDurationSeconds,
			"credits":                credits,
		},
	)
	if err != nil {
		return "", err
	}
	return result.ID.String(), nil
}

type BidsFetchResult struct {
	ID                   edgedb.UUID `json:"id" edgedb:"id"`
	AudioURI             string      `json:"audio_uri" edgedb:"audio_uri"`
	AudioDurationSeconds int64       `json:"audio_duration_seconds" edgedb:"audio_duration_seconds"`
	Credits              int64       `json:"credits" edgedb:"credits"`
	CreatedAt            time.Time   `json:"created_at" edgedb:"created_at"`
}

func BidsFetch(ctx context.Context, tx *edgedb.Tx) ([]BidsFetchResult, error) {
	result := []BidsFetchResult{}

	err := tx.Query(
		ctx,
		`SELECT Bid {
			id,
			audio_uri,
			audio_duration_seconds,
			credits,
			created_at
		}
		FILTER .user.identity = (global ext::auth::ClientTokenIdentity)
		ORDER BY .created_at DESC`,
		&result,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BidsTopFetchResult struct {
	ID                   edgedb.UUID `json:"id" edgedb:"id"`
	AudioDurationSeconds int64       `json:"audio_duration_seconds" edgedb:"audio_duration_seconds"`
	Credits              int64       `json:"credits" edgedb:"credits"`
	CreatedAt            time.Time   `json:"created_at" edgedb:"created_at"`
	User                 struct {
		ID       edgedb.UUID        `json:"id" edgedb:"id"`
		Username string             `json:"username" edgedb:"username"`
		ImageURI edgedb.OptionalStr `json:"image_uri" edgedb:"image_uri"`
	} `json:"user" edgedb:"user"`
}

func BidsTopFetch(ctx context.Context, tx *edgedb.Tx) ([]BidsTopFetchResult, error) {
	result := []BidsTopFetchResult{}

	err := tx.Query(
		ctx,
		`SELECT Bid {
			id,
			audio_duration_seconds,
			credits,
			created_at,
			user: {
				id,
				username,
				image_uri
			}
		}
		ORDER BY .credits / .audio_duration_seconds DESC THEN .created_at ASC`,
		&result,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BidsTopDequeueResult struct {
	edgedb.Optional
	ID                   edgedb.UUID `edgedb:"id"`
	AudioURI             string      `edgedb:"audio_uri"`
	AudioDurationSeconds int64       `edgedb:"audio_duration_seconds"`
	Credits              int64       `edgedb:"credits"`
	User                 struct {
		ID edgedb.UUID `edgedb:"id"`
	} `edgedb:"user"`
}

type BidsTopDequeueReturn struct {
	ID                   edgedb.UUID `json:"id"`
	AudioURI             string      `json:"audio_uri"`
	AudioDurationSeconds int64       `json:"audio_duration_seconds"`
	Credits              int64       `json:"credits"`
	UserID               edgedb.UUID `json:"user_id"`
}

func BidsTopDequeue(ctx context.Context, tx *edgedb.Tx) (*BidsTopDequeueReturn, error) {
	result := BidsTopDequeueResult{}

	err := tx.QuerySingle(
		ctx,
		`WITH
			bid := (
				DELETE Bid
				ORDER BY .credits / .audio_duration_seconds DESC THEN .created_at ASC
				LIMIT 1
			)
		SELECT bid {
			id,
			audio_uri,
			audio_duration_seconds,
			credits,
			user: {
				id
			}
		}`,
		&result,
	)
	if err != nil {
		return nil, err
	}
	if result.Missing() {
		return nil, nil
	}
	return &BidsTopDequeueReturn{
		ID:                   result.ID,
		AudioURI:             result.AudioURI,
		AudioDurationSeconds: result.AudioDurationSeconds,
		Credits:              result.Credits,
		UserID:               result.User.ID,
	}, nil
}

type BidDeleteResult struct {
	edgedb.Optional
	ID edgedb.UUID `edgedb:"id"`
}

func BidDelete(ctx context.Context, tx *edgedb.Tx, bidID edgedb.UUID) error {
	var result BidDeleteResult

	err := tx.QuerySingle(
		ctx,
		`DELETE Bid
		FILTER .id = <uuid>$bid_id AND .user.identity = (global ext::auth::ClientTokenIdentity)`,
		&result,
		map[string]interface{}{
			"bid_id": bidID,
		},
	)
	if err != nil {
		return err
	}
	if result.Missing() {
		return errors.New("bid does not exist")
	}
	return nil
}
