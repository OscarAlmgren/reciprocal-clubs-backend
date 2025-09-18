module reciprocal-clubs-backend/services/notification-service

go 1.25

require (
	github.com/gorilla/mux v1.8.1
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.9
	gorm.io/gorm v1.31.0
	reciprocal-clubs-backend/pkg/shared/config v0.0.0
	reciprocal-clubs-backend/pkg/shared/database v0.0.0
	reciprocal-clubs-backend/pkg/shared/logging v0.0.0
	reciprocal-clubs-backend/pkg/shared/messaging v0.0.0
	reciprocal-clubs-backend/pkg/shared/monitoring v0.0.0
)

require (
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nats-io/nats.go v1.45.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250908214217-97024824d090 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace reciprocal-clubs-backend/pkg/shared/auth => ../../pkg/shared/auth

replace reciprocal-clubs-backend/pkg/shared/config => ../../pkg/shared/config

replace reciprocal-clubs-backend/pkg/shared/database => ../../pkg/shared/database

replace reciprocal-clubs-backend/pkg/shared/errors => ../../pkg/shared/errors

replace reciprocal-clubs-backend/pkg/shared/logging => ../../pkg/shared/logging

replace reciprocal-clubs-backend/pkg/shared/messaging => ../../pkg/shared/messaging

replace reciprocal-clubs-backend/pkg/shared/monitoring => ../../pkg/shared/monitoring

replace reciprocal-clubs-backend/pkg/shared/utils => ../../pkg/shared/utils
