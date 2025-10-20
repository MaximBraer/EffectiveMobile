package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

type router interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
}

type RouterSuite struct {
	suite.Suite

	Router router

	addr string
	wg   sync.WaitGroup
}

func (s *RouterSuite) SetupSuite() {
	if s.Router == nil {
		s.FailNow("router is not set")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.FailNow(fmt.Sprintf("failed to create listener: %v", err))
	}

	s.addr = listener.Addr().String()

	s.wg = sync.WaitGroup{}
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		err := s.Router.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Require().NoError(err) //nolint: testifylint
		}
	}()
}

func (s *RouterSuite) TearDownSuite() {
	const defaultTimeout = 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err := s.Router.Shutdown(ctx)
	s.Require().NoError(err)

	s.wg.Wait()
}

func (s *RouterSuite) MakeURL(path string) string {
	return fmt.Sprintf("http://%s/%s", s.addr, path)
}

func (s *RouterSuite) PostJSONWithHeadersResponse(path, body string, headers http.Header) *http.Response {
	URL, err := url.Parse(s.MakeURL(path))
	s.Require().NoError(err)

	if headers == nil {
		headers = make(http.Header)
	}

	headers["Content-Type"] = []string{"application/json"}

	client := http.Client{}

	request := &http.Request{
		Method: http.MethodPost,
		URL:    URL,
		Header: headers,
		Body:   io.NopCloser(bytes.NewBufferString(body)),
	}

	resp, err := client.Do(request)
	s.Require().NoError(err)

	return resp
}

func (s *RouterSuite) GetWithResponse(path string) *http.Response {
	URL, err := url.Parse(s.MakeURL(path))
	s.Require().NoError(err)

	client := http.Client{}
	resp, err := client.Do(&http.Request{Method: http.MethodGet, URL: URL})
	s.Require().NoError(err)

	return resp
}

func (s *RouterSuite) PutJSONWithHeadersResponse(path, body string, headers http.Header) *http.Response {
	URL, err := url.Parse(s.MakeURL(path))
	s.Require().NoError(err)

	if headers == nil {
		headers = make(http.Header)
	}

	headers["Content-Type"] = []string{"application/json"}

	client := http.Client{}

	request := &http.Request{
		Method: http.MethodPut,
		URL:    URL,
		Header: headers,
		Body:   io.NopCloser(bytes.NewBufferString(body)),
	}

	resp, err := client.Do(request)
	s.Require().NoError(err)

	return resp
}
