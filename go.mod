module github.com/CMSgov/bcda-app

go 1.15

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/aws/aws-sdk-go v1.28.8
	github.com/bgentry/que-go v1.0.1
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.1-0.20180309185540-3c771ce311b7+incompatible
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/dgryski/go-lttb v0.0.0-20180810165845-318fcdf10a77 // indirect
	github.com/dimchansky/utfbom v1.1.1
	github.com/go-chi/chi v4.0.3-0.20190508141739-08c92af09aaf+incompatible
	github.com/go-chi/render v1.0.1
	github.com/go-gota/gota v0.10.1
	github.com/google/fhir/go v0.0.0-20210120234235-b7cfb32dc82f
	github.com/google/uuid v1.1.2
	github.com/huandu/go-sqlbuilder v1.10.0
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.1.1-0.20180608201956-39bbc98d99d7+incompatible
	github.com/lib/pq v1.9.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mitchellh/mapstructure v1.2.3
	github.com/newrelic/go-agent/v3 v3.9.0
	github.com/newrelic/go-agent/v3/integrations/nrlogrus v1.0.0
	github.com/otiai10/copy v1.4.2
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/viper v1.7.1
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tsenart/go-tsz v0.0.0-20180814235614-0bd30b3df1c3 // indirect
	github.com/tsenart/vegeta v12.1.0+incompatible
	github.com/urfave/cli v1.20.1-0.20180226030253-8e01ec4cd3e2
	golang.org/x/crypto v0.0.0-20200311171314-f7b00557c8c4
	google.golang.org/genproto v0.0.0-20201204160425-06b3db808446 // indirect
	google.golang.org/grpc v1.34.0 // indirect
	google.golang.org/grpc/examples v0.0.0-20210111180913-4cf4a98505bc // indirect
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
