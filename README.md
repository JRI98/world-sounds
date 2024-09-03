# World Sounds

Platform where people can bid to stream audio to everyone listening. Uses [EdgeDB](https://www.edgedb.com/) as the database, [MinIO](https://min.io/) as the file storage and [Paddle](https://www.paddle.com/) as payments processor.

## Database

### Migrations

- `edgedb migration create`
- `edgedb migrate`

### Queries

- `go install github.com/edgedb/edgedb-go/cmd/edgeql-go@latest`
- Write your query in models/queries.edgeql
- `go generate models/models.go`
