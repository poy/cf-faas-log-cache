package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/poy/cf-faas-log-cache/internal/web"
)

type CapiClient interface {
	SetEnvironmentVariables(ctx context.Context, appGuid string, vars map[string]string) error
	GetEnvironmentVariables(ctx context.Context, appGuid string) (map[string]string, error)
	Restart(ctx context.Context, appGuid string) error
}

type Saver struct {
	appGuid string
	c       CapiClient
	log     *log.Logger
}

func NewSaver(appGuid string, c CapiClient, log *log.Logger) *Saver {
	return &Saver{
		appGuid: appGuid,
		c:       c,
		log:     log,
	}
}

func (s *Saver) SaveState(ctx context.Context, qs []web.Query) error {
	data, err := json.Marshal(struct {
		Queries []web.Query `json:"queries"`
	}{Queries: qs})

	if err != nil {
		s.log.Panicf("failed to marshal queries: %s", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.c.SetEnvironmentVariables(ctx, s.appGuid, map[string]string{"QUERIES": string(data)}); err != nil {
		return fmt.Errorf("setting env vars failed: %s", err)
	}

	if err := s.c.Restart(ctx, s.appGuid); err != nil {
		return fmt.Errorf("restarting app failed: %s", err)
	}

	return nil
}
