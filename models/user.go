package models

import (
	"context"
	"errors"
	"time"

	"github.com/anandvarma/namegen"
	"github.com/edgedb/edgedb-go"
)

type UserCreateResult struct {
	ID edgedb.UUID `edgedb:"id"`
}

func UserCreate(ctx context.Context, tx *edgedb.Tx) error {
	generator := namegen.NewWithPostfixId([]namegen.DictType{namegen.Adjectives, namegen.Colors, namegen.Animals}, namegen.Numeric, 4)
	username := generator.Get()

	var result UserCreateResult

	err := tx.QuerySingle(
		ctx,
		`INSERT User {
			username := <str>$username,
			credits := 0,
			identity := (global ext::auth::ClientTokenIdentity)
		}`,
		&result,
		map[string]interface{}{
			"username": username,
		})
	if err != nil {
		return err
	}
	return nil
}

type UserFetchResult struct {
	edgedb.Optional
	ID        edgedb.UUID        `json:"id" edgedb:"id"`
	Username  string             `json:"username" edgedb:"username"`
	Credits   int64              `json:"credits" edgedb:"credits"`
	ImageURI  edgedb.OptionalStr `json:"image_uri" edgedb:"image_uri"`
	CreatedAt time.Time          `json:"created_at" edgedb:"created_at"`
}

func UserFetch(ctx context.Context, tx *edgedb.Tx) (*UserFetchResult, error) {
	var result UserFetchResult

	err := tx.QuerySingle(
		ctx,
		`SELECT User {
			id,
			username,
			credits,
			image_uri,
			created_at
		}
		FILTER .identity = (global ext::auth::ClientTokenIdentity)`,
		&result,
	)
	if err != nil {
		return nil, err
	}
	if result.Missing() {
		return nil, nil
	}
	return &result, nil
}

type UserDecrementCreditsResult struct {
	edgedb.Optional
	ID edgedb.UUID `edgedb:"id"`
}

func UserDecrementCredits(ctx context.Context, tx *edgedb.Tx, amount int64) error {
	var result UserDecrementCreditsResult

	err := tx.QuerySingle(
		ctx,
		`UPDATE User
		FILTER .identity = (global ext::auth::ClientTokenIdentity)
		SET {
			credits := .credits - <int64>$amount
		}`,
		&result,
		map[string]interface{}{
			"amount": amount,
		},
	)
	if err != nil {
		return err
	}
	if result.Missing() {
		return errors.New("user does not exist")
	}
	return nil
}

type UserIncrementCreditsResult struct {
	edgedb.Optional
	ID edgedb.UUID `edgedb:"id"`
}

func UserIncrementCredits(ctx context.Context, tx *edgedb.Tx, userID edgedb.UUID, amount int64) error {
	var result UserIncrementCreditsResult

	err := tx.QuerySingle(
		ctx,
		`UPDATE User
		FILTER .id = <uuid>$user_id
		SET {
			credits := .credits + <int64>$amount
		}`,
		&result,
		map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
		},
	)
	if err != nil {
		return err
	}
	if result.Missing() {
		return errors.New("user does not exist")
	}
	return nil
}

type UserUpdateResult struct {
	edgedb.Optional
	ID edgedb.UUID `edgedb:"id"`
}

func UserUpdate(ctx context.Context, tx *edgedb.Tx, username *string) error {
	var result UserIncrementCreditsResult

	err := tx.QuerySingle(
		ctx,
		`UPDATE User
		FILTER .identity = (global ext::auth::ClientTokenIdentity)
		SET {
			username := <optional str>$username
		}`,
		&result,
		map[string]interface{}{
			"username": stringPointerToOptionalStr(username),
		},
	)
	if err != nil {
		return err
	}
	if result.Missing() {
		return errors.New("user does not exist")
	}
	return nil
}

type UserUpdateImageResult struct {
	edgedb.Optional
	ID edgedb.UUID `edgedb:"id"`
}

func UserUpdateImage(ctx context.Context, tx *edgedb.Tx, imageURI string) error {
	var result UserIncrementCreditsResult

	err := tx.QuerySingle(
		ctx,
		`UPDATE User
		FILTER .identity = (global ext::auth::ClientTokenIdentity)
		SET {
			image_uri := <str>$image_uri
		}`,
		&result,
		map[string]interface{}{
			"image_uri": imageURI,
		},
	)
	if err != nil {
		return err
	}
	if result.Missing() {
		return errors.New("user does not exist")
	}
	return nil
}
