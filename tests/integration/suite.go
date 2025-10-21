package integration

import (
	"net/http"
	"time"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

func (s *Suite) SetupTest() {
	s.waitForServiceReady(mainHost)
}

func (s *Suite) waitForServiceReady(host string) {
	const (
		eventTimeout = 20 * time.Second
		eventTick    = 100 * time.Millisecond
	)

    s.Eventually(func() bool {
        _, _, err := doRequest(http.MethodGet, host, "/health", nil, nil, nil) //nolint:bodyclose

		return err == nil
	}, eventTimeout, eventTick)
}

func (s *Suite) Eventually(
	condition func() bool,
	waitFor time.Duration,
	tick time.Duration,
	msgAndArgs ...any,
) bool {
	s.T().Helper()

	if condition() {
		return true
	}

	return s.Suite.Eventually(condition, waitFor, tick, msgAndArgs...)
}
