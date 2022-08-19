module github.com/CMSgov/bcda-app

go 1.18

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/aws/aws-sdk-go v1.44.47
	github.com/bgentry/que-go v1.0.1
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/dgrijalva/jwt-go v3.2.1-0.20180309185540-3c771ce311b7+incompatible
	github.com/dimchansky/utfbom v1.1.1
	github.com/go-chi/chi v5.0.7+incompatible
	github.com/go-chi/render v1.0.1
	github.com/go-gota/gota v0.12.0
	github.com/go-testfixtures/testfixtures/v3 v3.5.0
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/fhir/go v0.0.0-20220518004845-30f5cde7c5cd
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/howeyc/fsnotify v0.9.0
	github.com/huandu/go-sqlbuilder v1.14.1
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/mattn/go-colorable v0.1.12
	github.com/mitchellh/mapstructure v1.5.0
	github.com/newrelic/go-agent/v3 v3.17.0
	github.com/newrelic/go-agent/v3/integrations/nrlogrus v1.0.0
	github.com/otiai10/copy v1.7.0
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/securego/gosec v0.0.0-20200401082031-e946c8c39989
	github.com/sirupsen/logrus v1.8.1
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/tsenart/vegeta v12.7.0+incompatible
	github.com/urfave/cli v1.22.9
	github.com/xo/usql v0.8.2
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	golang.org/x/text v0.3.7
	gotest.tools/gotestsum v1.6.2
)

require (
	bitbucket.org/creachadair/stringset v0.0.10 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/c2h5oh/datasize v0.0.0-20200825124411-48ed595a09d2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/dgryski/go-lttb v0.0.0-20180810165845-318fcdf10a77 // indirect
	github.com/frankban/quicktest v1.14.3 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lib/pq v1.10.6 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/tsenart/go-tsz v0.0.0-20180814235614-0bd30b3df1c3 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/net v0.0.0-20220630215102-69896b714898 // indirect
	golang.org/x/sys v0.0.0-20220627191245-f75cf1eec38b // indirect
	gonum.org/v1/gonum v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20220630174209-ad1d48641aa7 // indirect
	google.golang.org/grpc v1.47.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-chi/chi => github.com/go-chi/chi v4.0.3-0.20190508141739-08c92af09aaf+incompatible
