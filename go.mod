module github.com/kooshapari/bifrost-extensions

go 1.24.3

require (
	github.com/coder/agentapi v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.2
	github.com/google/uuid v1.6.0
	github.com/hatchet-dev/hatchet v0.73.73
	github.com/jackc/pgx/v5 v5.7.6
	github.com/kooshapari/CLIProxyAPI/v7 v7.0.0
	github.com/KooshaPari/phenotype-go-kit v0.0.0
	github.com/maximhq/bifrost/core v1.2.30
	github.com/nats-io/nats.go v1.47.0
	github.com/neo4j/neo4j-go-driver/v5 v5.28.4
	github.com/pgvector/pgvector-go v0.3.0
	github.com/redis/go-redis/v9 v9.17.1
	github.com/spf13/viper v1.18.2
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.13 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.7 // indirect
	github.com/aws/smithy-go v1.23.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.14.1 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mark3labs/mcp-go v0.41.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.67.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Development: use local paths until packages are published
replace github.com/maximhq/bifrost/core => ../../bifrost/core

replace github.com/coder/agentapi => ../../agentapi-plusplus

replace github.com/kooshapari/CLIProxyAPI/v7 => ../../CLIProxyAPI

replace github.com/KooshaPari/phenotype-go-kit => ../../template-commons/phenotype-go-kit
