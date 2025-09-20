module reciprocal-clubs-backend/services/api-gateway

go 1.25

require (
	github.com/99designs/gqlgen v0.17.45
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/rs/cors v1.10.1
	google.golang.org/grpc v1.75.1
	reciprocal-clubs-backend/pkg/shared/auth v0.0.0
	reciprocal-clubs-backend/pkg/shared/config v0.0.0
	reciprocal-clubs-backend/pkg/shared/logging v0.0.0
	reciprocal-clubs-backend/pkg/shared/monitoring v0.0.0
	reciprocal-clubs-backend/pkg/shared/utils v0.0.0
)

require (
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sosodev/duration v1.2.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.11
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250908214217-97024824d090 // indirect
	google.golang.org/protobuf v1.36.9
)

replace reciprocal-clubs-backend/pkg/shared/auth => ../../pkg/shared/auth

replace reciprocal-clubs-backend/pkg/shared/config => ../../pkg/shared/config

replace reciprocal-clubs-backend/pkg/shared/logging => ../../pkg/shared/logging

replace reciprocal-clubs-backend/pkg/shared/monitoring => ../../pkg/shared/monitoring

replace reciprocal-clubs-backend/pkg/shared/utils => ../../pkg/shared/utils
