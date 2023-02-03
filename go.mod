module github.com/mainflux/mainflux

go 1.19

require (
	github.com/caarlos0/env/v7 v7.0.0
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/docker/docker v20.10.21+incompatible
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/fatih/color v1.13.0
	github.com/fiorix/go-smpp v0.0.0-20210403173735-2894b96e70ba
	github.com/go-kit/kit v0.12.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-zoo/bone v1.3.0
	github.com/gocql/gocql v1.2.1
	github.com/gofrs/uuid v4.3.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/golang/protobuf v1.5.2
	github.com/gopcua/opcua v0.1.6
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/vault/api v1.8.1
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f
	github.com/influxdata/influxdb v1.10.0
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa
	github.com/jackc/pgtype v1.13.0
	github.com/jackc/pgx/v5 v5.2.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/mainflux/mproxy v0.2.3
	github.com/mainflux/senml v1.5.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/nats-io/nats.go v1.18.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/ory/dockertest/v3 v3.9.1
	github.com/ory/keto/proto/ory/keto/acl/v1alpha1 v0.0.0-20210616104402-80e043246cf9
	github.com/pelletier/go-toml v1.9.5
	github.com/plgd-dev/go-coap/v2 v2.6.0
	github.com/prometheus/client_golang v1.13.0
	github.com/rabbitmq/amqp091-go v1.5.0
	github.com/rubenv/sql-migrate v1.2.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.1
	github.com/subosito/gotenv v1.4.1
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	go.mongodb.org/mongo-driver v1.10.3
	golang.org/x/crypto v0.1.0
	golang.org/x/net v0.1.0
	golang.org/x/sync v0.1.0
	gonum.org/v1/gonum v0.12.0
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v20.10.21+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.3.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.5 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.7 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/sdk v0.6.0 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.0.0-20220808134915-39b0c02b01ae // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pion/dtls/v2 v2.1.5 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/transport v0.13.1 // indirect
	github.com/pion/udp v0.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.1 // indirect
	github.com/xdg-go/stringprep v1.0.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/mod v0.6.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	golang.org/x/tools v0.2.0 // indirect
	google.golang.org/genproto v0.0.0-20221025140454-527a21cfbd71 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
