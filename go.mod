module github.com/CMSgov/bcda-app

go 1.15

require (
	cloud.google.com/go v0.38.0 // indirect
	contrib.go.opencensus.io/exporter/jaeger v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.11.0 // indirect
	contrib.go.opencensus.io/resource v0.1.1 // indirect
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/DataDog/zstd v1.3.5 // indirect
	github.com/alexbrainman/sspi v0.0.0-20180613141037-e580b900e9f5 // indirect
	github.com/aws/aws-sdk-go v1.21.3
	github.com/bgentry/que-go v1.0.1
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/boj/redistore v0.0.0-20160128113310-fc113767cd6b // indirect
	github.com/buger/jsonparser v0.0.0-20180318095312-2cac668e8456 // indirect
	github.com/campoy/embedmd v0.0.0-20181127031020-97c13d6e4160 // indirect
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cncf/udpa/go v0.0.0-20200629203442-efcf912fb354 // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/corpix/uarand v0.0.0-20170903190822-2b8494104d86 // indirect
	github.com/dgrijalva/jwt-go v3.2.1-0.20180309185540-3c771ce311b7+incompatible
	github.com/dgryski/go-gk v0.0.0-20200319235926-a69029f61654 // indirect
	github.com/dgryski/go-lttb v0.0.0-20180810165845-318fcdf10a77 // indirect
	github.com/dlclark/regexp2 v1.1.6 // indirect
	github.com/dop251/goja v0.0.0-20180304123926-9183045acc25 // indirect
	github.com/envoyproxy/go-control-plane v0.9.5 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/gin-gonic/contrib v0.0.0-20180614032058-39cfb9727134 // indirect
	github.com/gin-gonic/gin v0.0.0-20181126150151-b97ccf3a43d2 // indirect
	github.com/go-chi/chi v4.0.3-0.20190508141739-08c92af09aaf+incompatible
	github.com/go-chi/render v1.0.1
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gorilla/sessions v1.2.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/icrowley/fake v0.0.0-20180203215853-4178557ae428 // indirect
	github.com/influxdata/tdigest v0.0.0-20181121200506-bf2b5ad3c0a9 // indirect
	github.com/itsjamie/gin-cors v0.0.0-20160420130702-97b4a9da7933 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.1.1-0.20180608201956-39bbc98d99d7+incompatible
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/juju/errors v0.0.0-20170703010042-c7d06af17c68 // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20190613124551-e81189438503 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-isatty v0.0.8 // indirect
	github.com/mitre/heart v0.0.0-20160825192324-0c46b433a490 // indirect
	github.com/newrelic/go-agent v3.9.0+incompatible
	github.com/newrelic/go-agent/v3 v3.9.0
	github.com/otiai10/copy v1.2.0
	github.com/pborman/uuid v0.0.0-20180122190007-c65b2f87fee3
	github.com/pebbe/util v0.0.0-20140716220158-e0e04dfe647c // indirect
	github.com/pkg/errors v0.8.1
	github.com/samply/golang-fhir-models/fhir-models v0.2.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.2.3-0.20181002233221-2db35c88b92a
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/tsenart/go-tsz v0.0.0-20180814235614-0bd30b3df1c3 // indirect
	github.com/tsenart/vegeta v12.1.0+incompatible
	github.com/ugorji/go v1.1.5-pre // indirect
	github.com/urfave/cli v1.20.1-0.20180226030253-8e01ec4cd3e2
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.1.4 // indirect
	go.opencensus.io v0.22.0 // indirect
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422 // indirect
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	golang.org/x/tools v0.0.0-20190628021728-85b1a4bcd4e6 // indirect
	google.golang.org/api v0.5.0 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20201204160425-06b3db808446 // indirect
	google.golang.org/grpc v1.34.0-dev // indirect
	google.golang.org/grpc/examples v0.0.0-20210111180913-4cf4a98505bc // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/square/go-jose.v1 v1.1.1 // indirect
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	gorm.io/gorm v1.20.8
)

replace github.com/opencensus-integrations/gomongowrapper => github.com/eug48/gomongowrapper v0.0.3
