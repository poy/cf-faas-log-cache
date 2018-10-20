package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	"github.com/poy/cf-faas-log-cache/internal/promql"
	"github.com/poy/cf-faas-log-cache/internal/state"
	"github.com/poy/cf-faas-log-cache/internal/web"
	pkgpromql "github.com/poy/cf-faas-log-cache/pkg/promql"
	gocapi "github.com/poy/go-capi"
)

func main() {
	log := log.New(os.Stderr, "", log.LstdFlags)
	log.Printf("starting cf-faas-log-cache...")
	defer log.Printf("closing cf-faas-log-cache...")

	cfg := loadConfig(log)

	capiClient := gocapi.NewClient(
		cfg.VcapApplication.CAPIAddr,
		cfg.VcapApplication.ApplicationID,
		cfg.VcapApplication.SpaceID,
		http.DefaultClient,
	)

	sanitizer := pkgpromql.NewSanitizer(capiClient)

	logCacheClient := pkgpromql.NewClient(
		cfg.VcapApplication.LogCacheAddr,
		sanitizer,
		http.DefaultClient,
	)

	stateSaver := state.NewSaver(cfg.VcapApplication.ApplicationID, capiClient, log)
	resolver := web.NewResolver(stateSaver, log)

	go func() {
		var readers []*promql.Reader
		for _, q := range cfg.Queries.Queries {
			q.Path = "http://" + cfg.CFFaasAddr + q.Path
			readers = append(readers, promql.NewReader(q, logCacheClient, http.DefaultClient, log))
		}

		for range time.Tick(cfg.Interval) {
			for _, r := range readers {
				r.Tick()
			}
		}
	}()

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), resolver); err != nil {
		log.Fatal(err)
	}
}

type config struct {
	Port            int             `env:"PORT,required,report"`
	VcapApplication vcapApplication `env:"VCAP_APPLICATION, required, report"`
	Queries         Queries         `env:"QUERIES, report"`
	Interval        time.Duration   `env:"INTERVAL,report"`
	CFFaasAddr      string          `env:"CF_FAAS_ADDR,required,report"`

	SkipSSLValidation bool `env:"SKIP_SSL_VALIDATION, report"`
}

type vcapApplication struct {
	CAPIAddr      string `json:"cf_api"`
	LogCacheAddr  string // Inferred from CAPIAddr
	ApplicationID string `json:"application_id"`
	SpaceID       string `json:"space_id"`
}

func (a *vcapApplication) UnmarshalEnv(data string) error {
	if err := json.Unmarshal([]byte(data), a); err != nil {
		return err
	}

	a.CAPIAddr = strings.Replace(a.CAPIAddr, "https", "http", 1)
	a.LogCacheAddr = strings.Replace(a.CAPIAddr, "api", "log-cache", 1)
	return nil
}

type Queries struct {
	Queries []web.Query `json:"queries"`
}

func (q *Queries) UnmarshalEnv(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), q)
}

func loadConfig(log *log.Logger) config {
	cfg := config{
		Interval: time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	// Use HTTP so we can use HTTP_PROXY
	cfg.VcapApplication.CAPIAddr = strings.Replace(cfg.VcapApplication.CAPIAddr, "https", "http", 1)

	envstruct.WriteReport(&cfg)
	return cfg
}
