package models

import (
	"context"
	"time"

	"github.com/edgedb/edgedb-go"
)

type StreamFetchResult struct {
	ID                   edgedb.UUID `json:"id" edgedb:"id"`
	AudioUri             string      `json:"audio_uri" edgedb:"audio_uri"`
	AudioDurationSeconds int64       `json:"audio_duration_seconds" edgedb:"audio_duration_seconds"`
	Credits              int64       `json:"credits" edgedb:"credits"`
	CreatedAt            time.Time   `json:"created_at" edgedb:"created_at"`
}

func StreamFetch(ctx context.Context, tx *edgedb.Tx) ([]StreamFetchResult, error) {
	result := []StreamFetchResult{}

	err := tx.Query(
		ctx,
		`SELECT Stream {
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

type StreamLatestFetchResult struct {
	ID                   edgedb.UUID `json:"id" edgedb:"id"`
	AudioURI             string      `json:"audio_uri" edgedb:"audio_uri"`
	AudioDurationSeconds int64       `json:"audio_duration_seconds" edgedb:"audio_duration_seconds"`
	Credits              int64       `json:"credits" edgedb:"credits"`
	User                 struct {
		ID       edgedb.UUID        `json:"id" edgedb:"id"`
		Username string             `json:"username" edgedb:"username"`
		ImageURI edgedb.OptionalStr `json:"image_uri" edgedb:"image_uri"`
	} `json:"user" edgedb:"user"`
	CreatedAt time.Time `json:"created_at" edgedb:"created_at"`
}

func StreamLastestFetch(ctx context.Context, tx *edgedb.Tx, offset int64, limit int64) ([]StreamLatestFetchResult, error) {
	result := []StreamLatestFetchResult{}

	err := tx.Query(
		ctx,
		`SELECT Stream {
			id,
			audio_uri,
			audio_duration_seconds,
			credits,
			user: {
				id,
				username,
				image_uri
			},
			created_at
		}
		ORDER BY .created_at DESC
		OFFSET <int64>$offset
		LIMIT <int64>$limit`,
		&result,
		map[string]any{
			"offset": offset,
			"limit":  limit,
		},
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type StreamCreateResult struct {
	ID edgedb.UUID `edgedb:"id"`
}

func StreamCreate(ctx context.Context, tx *edgedb.Tx, audioURI string, audioDurationSeconds int64, credits int64, userID edgedb.UUID) (string, error) {
	var result StreamCreateResult

	err := tx.QuerySingle(
		ctx,
		`INSERT Stream {
			audio_uri := <str>$audio_uri,
			audio_duration_seconds := <int64>$audio_duration_seconds,
			credits := <int64>$credits,
			user := (
				select User
				filter .id = <uuid>$user_id
			)
		}`,
		&result,
		map[string]interface{}{
			"audio_uri":              audioURI,
			"audio_duration_seconds": audioDurationSeconds,
			"credits":                credits,
			"user_id":                userID,
		},
	)
	if err != nil {
		return "", err
	}
	return result.ID.String(), nil
}
