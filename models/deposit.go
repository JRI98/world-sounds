package models

import (
	"context"
	"time"

	"github.com/edgedb/edgedb-go"
)

type DepositsFetchResult struct {
	ID        edgedb.UUID `json:"id" edgedb:"id"`
	Credits   int64       `json:"credits" edgedb:"credits"`
	CreatedAt time.Time   `json:"created_at" edgedb:"created_at"`
}

func DepositsFetch(ctx context.Context, tx *edgedb.Tx) ([]DepositsFetchResult, error) {
	result := []DepositsFetchResult{}

	err := tx.Query(
		ctx,
		`SELECT Deposit {
			id,
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

type DepositCreateResult struct {
	ID edgedb.UUID `edgedb:"id"`
}

func DepositCreate(ctx context.Context, tx *edgedb.Tx, credits int64, info []byte, remoteTransactionID string, userID edgedb.UUID) (string, error) {
	var result DepositCreateResult

	err := tx.QuerySingle(
		ctx,
		`INSERT Deposit {
			credits := <int64>$credits,
			info := <json>$info,
			remote_transaction_id := <str>$remote_transaction_id,
			user := (
				SELECT User
				FILTER .id = <uuid>$user_id
			)
		}`,
		&result,
		map[string]interface{}{
			"credits":               credits,
			"info":                  info,
			"remote_transaction_id": remoteTransactionID,
			"user_id":               userID,
		},
	)
	if err != nil {
		return "", err
	}
	return result.ID.String(), nil
}
