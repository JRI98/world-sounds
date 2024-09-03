//go:generate edgeql-go -mixedcaps
package models

import (
	"context"
	"os"

	"github.com/edgedb/edgedb-go"
)

func NewDBService() (*edgedb.Client, error) {
	ctx := context.Background()
	options := edgedb.Options{
		TLSOptions: edgedb.TLSOptions{
			SecurityMode: edgedb.TLSModeInsecure,
		},
	}

	dsn := os.Getenv("EDGEDB_DSN")

	var client *edgedb.Client
	var err error
	if dsn == "" {
		client, err = edgedb.CreateClient(ctx, options)
	} else {
		client, err = edgedb.CreateClientDSN(ctx, dsn, options)
	}

	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetTx(client *edgedb.Client, authToken *string) func(ctx context.Context, action edgedb.TxBlock) error {
	if authToken == nil {
		return client.Tx
	}
	return client.WithGlobals(map[string]interface{}{"ext::auth::client_token": *authToken}).Tx
}

func stringPointerToOptionalStr(s *string) edgedb.OptionalStr {
	var optionalStr edgedb.OptionalStr
	if s != nil {
		optionalStr = edgedb.NewOptionalStr(*s)
	}
	return optionalStr
}
