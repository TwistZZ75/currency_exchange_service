module gw-exchanger

go 1.25.6

require (
	github.com/jackc/pgx/v5 v5.11.0
	github.com/joho/godotenv v1.5.1
	google.golang.org/grpc v1.84.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

require (
	github.com/stretchr/testify v1.12.1
	proto_exchange v0.0.0
)

replace proto_exchange => ../proto-exchange
