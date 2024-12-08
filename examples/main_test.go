package examples_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/linkedservices"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/promutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

type AppConfig struct {
	Services *linkedservices.Config                `yaml:"linked-services" mapstructure:"linked-services" json:"linked-services"`
	Metrics  map[string]promutil.MetricGroupConfig `yaml:"metrics,omitempty" mapstructure:"metrics,omitempty" json:"metrics,omitempty"`
}

var yamlCfg = []byte(`
linked-services:
   cache:
     redis:
      - addr: "localhost:6379"
        key: ""
        db: 0
        ttl: "30m"
        poolSize: 0  #Maximum number of socket connections. 0 is Default (10) 
        maxRetries: -1 #Maximum number of retries before giving up. 0 is Default (3), -1 (not 0) disables retries.
        dialTimeout: 1000 #In Milliseconds, Dial timeout for establishing new connections. 0 is Default (5000) 
        readTimeout: 1500 #In Milliseconds, Timeout for socket reads. 0 is Default (3000)
        writeTimeout: 1000 #In Milliseconds, Timeout for socket writes. 0 is Default (readTimeout)
        idleTimeout: 5000 #In Milliseconds,  Amount of time after which client closes idle connections.Should be less than server's timeout. 0 isDefault (5 minutes)
        metrics:
          group-id: "redis"
          counter-id: "cache-counter"
          histogram-id: "cache-histogram"
   rest-client:
      timeout: "15s"
      skv: true
      trace-req-name: "smp-{op-name}"
      # retry-count:2
      # retry-wait-time: "200ms"
      # retry-max-wait-time: "1s"
      # retry-on-errors:
      #   - 500
   kafka:
      - broker-name: "default"
        # bootstrap-servers: "kafka1:9092,kafka2:9092,kafka3:9092"
        # Event-Hub: testgect.servicebus.windows.net:9093
        bootstrap-servers: "localhost:9092"
        # SSL, SASL_SSL, PLAIN
        security-protocol: SSL
        sasl:
          # Azure Event-Hub type of config
          # mechanisms: PLAIN
          # username: "$ConnectionString"
          # password: "Endpoint=sb://testgect.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=fIx/54sQHwdKbjmkQyAW5qVkaZf2Tyi7Vk8fPhK8SSI="
          ca-location: /Users/marioa.imperato/projects/tpm/certs/poste-cert-ca.pem
          mechanisms: SCRAM-SHA-512
          username: ap-00073.pk-00161.svc-anag-consumer-cliente
          password: Gy995Ub3
          skv: false

        #    ssl:
        #      ca-location: /Users/marioa.imperato/projects/tpm/certs/tprod-coll-trustore.pem
        #   consumer related configs
        consumer:
          enable-auto-commit: false
          isolation-level: read_committed
          max-poll-records: 500
          max-poll-timeout: 100
          auto-offset-reset: earliest
          session-timeout-ms: 30000
          fetch-min-bytes: 1
          fetch-max-bytes: 3000000
          delay: 2000
          max-retry: 1
        #   producer related configs
        producer:
          # not-any-more
          # enable-transactions: false
          acks: "all"
          max-timeout-ms: 100000
          delivery-timeout: 3s
          max-retries: 2
          async-delivery-metrics:
            group-id: "kafka-async-producer"
            counter-id: "messages"
        # not any-more
        # tick-interval: 400ms
        # exit:
        #   on-fail: true
        #   on-eof: false
   mongo-db:
     - name: default
       #   host: "mongodb://10.70.150.88:27017,10.70.150.78:27017"
       host: "mongodb://localhost:27017"
       db-name: "tpm_orchestra"
       #   user: env ----> K2M_MONGO_USER
       #   pwd:  env ----> K2M_MONGO_PWD
       # TLS or PLAIN
       security-protocol: PLAIN
       tls:
         skip-verify: true
       bulkWriteOrdered: true
       # Admitted values: 1, majority
       write-concern: majority
       write-timeout: 120s
       pool:
         min-conn: 1
         max-conn: 20
         max-wait-queue-size: 1000
         max-wait-time: 1000
         max-connection-idle-time: 30000
         max-connection-life-time: 6000000
       collections:
         - id: movies
           name: movies

metrics:
   activity:
     namespace: bpap_ricarica_postepay
     subsystem: activity
     collectors:
     - id: activity-counter
       name: counter
       help: numero richieste
       labels:
         - id: type
           name: type
           default-value: N/A
         - id: name
           name: name
           default-value: N/A
         - id: endpoint
           name: endpoint
           default-value: N/A
         - id: status-code
           name: status_code
           default-value: N/A
       type: counter
     - id: activity-duration
       name: duration
       help: durata lavorazione richiesta
       labels:
         - id: type
           name: type
           default-value: N/A
         - id: name
           name: name
           default-value: N/A
         - id: endpoint
           name: endpoint
           default-value: N/A
         - id: status-code
           name: status_code
           default-value: N/A
       type: histogram
       buckets:
         type: linear
         start: 0.5
         width-factor: 0.5
         count: 10
   request-activity:
     namespace: bpap_ricarica_postepay
     subsystem: request_activity
     collectors:
       - id: activity-counter
         name: counter
         help: numero richieste
         labels:
           - id: name
             name: name
             default-value: N/A
           - id: status-code
             name: status_code
             default-value: N/A
         type: counter
   redis:
     namespace: bpap_ricarica_postepay
     subsystem: response_cache
     collectors:
       - id: cache-counter
         name: counter
         help: numero operazioni
         labels:
           - id: status-code
             name: status_code
             default-value: N/A
           - id: operation
             name: operation
             default-value: N/A
         type: counter
       - id: cache-histogram
         name: duration
         help: durata lavorazione cache
         labels:
           - id: status-code
             name: status_code
             default-value: N/A
           - id: operation
             name: operation
             default-value: N/A
         type: histogram
         buckets:
           type: linear
           start: 0.5
           width-factor: 0.5
           count: 10
   
   har-reporting:
       namespace: bpap_ricarica_postepay
       subsystem: har
       collectors:
         - id: har-messages
           name: messages
           help: produzione log har
           type: counter
           labels:
             - id: status-code
               name: status
               default-value: 500
   kafka-async-producer:
     namespace: bpap_ricarica_postepay
     subsystem: async_kafka_delivery
     collectors:
       - id: messages
         name: messages
         help: produzione kafka-activity in modalità async
         type: counter
         labels:
           - id: status-code
             name: status
             default-value: 500
           - id: broker-name
             name: broker_name
             default-value: "ND"
           - id: topic-name
             name: topic_name
             default-value: "ND"
           - id: error-code
             name: error_code
             default-value: "ND"
`)

func TestMain(m *testing.M) {

	const semLogContext = "examples::test-main"
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	path, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg(semLogContext)
	}
	log.Info().Str("work-dir", path).Msg(semLogContext)

	cfg := AppConfig{}
	err = yaml.Unmarshal(yamlCfg, &cfg)
	if err != nil {
		log.Fatal().Err(err).Msg(semLogContext)
	}

	err = kzxform.InitializeKazaamRegistry()
	if nil != err {
		log.Fatal().Err(err).Msg(semLogContext)
	}

	err = linkedservices.InitRegistry(cfg.Services)
	if nil != err {
		log.Fatal().Err(err).Msg(semLogContext + " linked services initialization error")
	}
	defer linkedservices.Close()

	_, err = promutil.InitRegistry(cfg.Metrics)
	if nil != err {
		log.Fatal().Err(err).Msg(semLogContext + " metrics registry initialization error")
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}
