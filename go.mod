module github.com/patbaumgartner/watchtower

go 1.27.0

require (
github.com/containerd/errdefs v1.0.0
github.com/containrrr/shoutrrr v0.8.0
github.com/distribution/reference v0.6.0
github.com/docker/cli v29.2.0+incompatible
github.com/moby/docker-image-spec v1.3.1
github.com/moby/moby/api v1.55.0
github.com/moby/moby/client v0.5.1
github.com/onsi/ginkgo v1.16.5
github.com/onsi/gomega v1.42.1
github.com/prometheus/client_golang v1.24.1
github.com/robfig/cron v1.2.0
github.com/sirupsen/logrus v1.10.1
github.com/spf13/cobra v1.10.2
github.com/spf13/pflag v1.0.10
github.com/spf13/viper v1.21.0
github.com/stretchr/testify v1.12.0
)

require (
github.com/containerd/errdefs/pkg v0.3.0 // indirect
github.com/docker/go-connections v0.8.1 // indirect
github.com/felixge/httpsnoop v1.1.0 // indirect
github.com/go-logr/logr v1.4.4 // indirect
github.com/go-logr/stdr v1.2.2 // indirect
github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
go.opentelemetry.io/auto/sdk v1.2.1 // indirect
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.70.0 // indirect
go.opentelemetry.io/otel v1.45.0 // indirect
go.opentelemetry.io/otel/metric v1.45.0 // indirect
go.opentelemetry.io/otel/trace v1.45.0 // indirect
go.yaml.in/yaml/v3 v3.0.5 // indirect
golang.org/x/net v0.58.0 // indirect
)

require (
github.com/Microsoft/go-winio v0.6.2 // indirect
github.com/beorn7/perks v1.0.1 // indirect
github.com/cespare/xxhash/v2 v2.3.0 // indirect
github.com/docker/docker-credential-helpers v0.6.1 // indirect
github.com/docker/go-units v0.5.0 // indirect
github.com/fatih/color v1.19.0 // indirect
github.com/fsnotify/fsnotify v1.10.1 // indirect
github.com/google/go-cmp v0.7.0 // indirect
github.com/inconshreveable/mousetrap v1.1.0 // indirect
github.com/mattn/go-colorable v0.1.15 // indirect
github.com/mattn/go-isatty v0.0.24 // indirect
github.com/nxadm/tail v1.4.11 // indirect
github.com/opencontainers/go-digest v1.0.0 // indirect
github.com/opencontainers/image-spec v1.1.1 // indirect
github.com/pelletier/go-toml/v2 v2.4.3 // indirect
github.com/prometheus/client_model v0.6.2 // indirect
github.com/prometheus/common v0.70.1 // indirect
github.com/prometheus/procfs v0.21.1 // indirect
github.com/sagikazarmark/locafero v0.12.0 // indirect
github.com/spf13/afero v1.15.0 // indirect
github.com/spf13/cast v1.10.0 // indirect
github.com/stretchr/objx v0.5.3 // indirect
github.com/subosito/gotenv v1.6.0 // indirect
golang.org/x/sys v0.47.0 // indirect
golang.org/x/text v0.41.0
google.golang.org/protobuf v1.36.11 // indirect
gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Keep Docker Engine dependencies on the split Moby API and client modules.