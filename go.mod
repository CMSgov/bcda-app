module github.com/CMSgov/bcda-app

go 1.15

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/aws/aws-sdk-go v1.37.33
	github.com/bgentry/que-go v1.0.1
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/c2h5oh/datasize v0.0.0-20200825124411-48ed595a09d2 // indirect
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/dgrijalva/jwt-go v3.2.1-0.20180309185540-3c771ce311b7+incompatible
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/dgryski/go-lttb v0.0.0-20180810165845-318fcdf10a77 // indirect
	github.com/dimchansky/utfbom v1.1.1
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/go-chi/chi v4.0.3-0.20190508141739-08c92af09aaf+incompatible
	github.com/go-chi/render v1.0.1
	github.com/go-delve/delve v1.6.0
	github.com/go-gota/gota v0.10.1
	github.com/go-testfixtures/testfixtures/v3 v3.5.0
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/golang/protobuf v1.5.1 // indirect
	github.com/google/fhir/go v0.0.0-20210120234235-b7cfb32dc82f
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/howeyc/fsnotify v0.9.0
	github.com/huandu/go-sqlbuilder v1.10.0
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/lib/pq v1.10.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8
	github.com/mitchellh/mapstructure v1.4.1
	github.com/newrelic/go-agent/v3 v3.9.0
	github.com/newrelic/go-agent/v3/integrations/nrlogrus v1.0.0
	github.com/otiai10/copy v1.4.2
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/securego/gosec v0.0.0-20200401082031-e946c8c39989
	github.com/sirupsen/logrus v1.8.1
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/viper v1.7.1
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tsenart/go-tsz v0.0.0-20180814235614-0bd30b3df1c3 // indirect
	github.com/tsenart/vegeta v12.7.0+incompatible
	github.com/urfave/cli v1.22.5
	github.com/xo/usql v0.8.2
	golang.org/x/crypto v0.0.0-20210317152858-513c2a44f670
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4 // indirect
	golang.org/x/sys v0.0.0-20210317225723-c4fcb01b228e // indirect
	google.golang.org/genproto v0.0.0-20210318145829-90b20ab00860 // indirect
	google.golang.org/grpc/examples v0.0.0-20210111180913-4cf4a98505bc // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/gotestsum v1.6.2
)

replace (
	github.com/aws/aws-sdk-go => github.com/aws/aws-sdk-go v1.21.3
	github.com/bgentry/que-go => github.com/bgentry/que-go v1.0.1
	github.com/cenkalti/backoff/v4 => github.com/cenkalti/backoff/v4 v4.0.2
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v3.2.1-0.20180309185540-3c771ce311b7+incompatible
	github.com/go-chi/chi => github.com/go-chi/chi v4.0.3-0.20190508141739-08c92af09aaf+incompatible
	github.com/go-chi/render => github.com/go-chi/render v1.0.1
	github.com/jackc/pgx => github.com/jackc/pgx v3.1.1-0.20180608201956-39bbc98d99d7+incompatible
	github.com/lib/pq => github.com/lib/pq v1.9.0
	github.com/newrelic/go-agent/v3 => github.com/newrelic/go-agent/v3 v3.9.0
	github.com/opencensus-integrations/gomongowrapper => github.com/eug48/gomongowrapper v0.0.3
	github.com/pborman/uuid => github.com/pborman/uuid v0.0.0-20180122190007-c65b2f87fee3
	github.com/pkg/errors => github.com/pkg/errors v0.8.0
	github.com/sirupsen/logrus => github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.4
	github.com/tsenart/vegeta => github.com/tsenart/vegeta v12.1.0+incompatible
	github.com/urfave/cli => github.com/urfave/cli v1.20.1-0.20180226030253-8e01ec4cd3e2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734
)
