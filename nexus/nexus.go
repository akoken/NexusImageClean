package nexus

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

const (
	RpcEndpoint = "service/extdirect"
	Asset       = "asset"
	Folder      = "folder"
	NotFount    = "No found data"
)

// ServerInfo contains the information needed to connect to a Nexus server
type ServerInfo struct {
	Host       string `json:"host" toml:"host"`
	Username   string `json:"username" toml:"username"`
	Password   string `json:"password" toml:"password"`
	Repository string `json:"repository" toml:"repository"`
}

func New(info ServerInfo) (Client, error) {
	c := new(DefaultClient)
	c.Host = info.Host
	c.Username = info.Username
	c.Password = info.Password
	c.Repository = info.Repository
	err := c.TestConn()
	return c, err
}

// Client is the interface which allows interacting with an IQ server
type Client interface {
	NewRequest(method, endpoint string, payload io.Reader) (*http.Request, error)
	Do(request *http.Request) ([]byte, *http.Response, error)
	Get(endpoint string) ([]byte, *http.Response, error)
	Post(endpoint string, payload io.Reader) ([]byte, *http.Response, error)
	Put(endpoint string, payload io.Reader) ([]byte, *http.Response, error)
	Del(endpoint string) (*http.Response, error)
	GetRepository(version string, direction ...string) (*RepositoryResponse, error)
	GetRepositoryAllTags(nodeID string, direction ...string) (*RepositoryTagsResponse, error)
	GetRepositoryTagInfo(tagID string) (*RepositoryTagInfoResponse, error)
	DeleteImageTag(name string) error
	Info() ServerInfo
	SetDebug(enable bool)
}

// DefaultClient provides an HTTP wrapper with optimized for communicating with a Nexus server
type DefaultClient struct {
	ServerInfo
	Debug bool
}

// NewRequest created an http.Request object based on an endpoint and fills in basic auth
func (s *DefaultClient) NewRequest(method, endpoint string, payload io.Reader) (request *http.Request, err error) {
	url := fmt.Sprintf("%s/%s", s.Host, endpoint)
	request, err = http.NewRequest(method, url, payload)
	if err != nil {
		return
	}

	request.SetBasicAuth(s.Username, s.Password)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return
}

// Do performs an http.Request and reads the body if StatusOK
func (s *DefaultClient) Do(request *http.Request) (body []byte, resp *http.Response, err error) {
	if s.Debug {
		dump, _ := httputil.DumpRequest(request, true)
		log.Println("debug: http request:")
		log.Printf("%q\n", dump)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err = client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// TODO: Trying to decide if this is a horrible idea or just kinda bad
	if resp.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(resp.Body)
		return
	}

	err = errors.New(resp.Status)
	return
}

func (s *DefaultClient) http(method, endpoint string, payload io.Reader) ([]byte, *http.Response, error) {
	request, err := s.NewRequest(method, endpoint, payload)
	if err != nil {
		return nil, nil, err
	}

	return s.Do(request)
}

// Get performs an HTTP GET against the indicated endpoint
func (s *DefaultClient) Get(endpoint string) ([]byte, *http.Response, error) {
	return s.http(http.MethodGet, endpoint, nil)
}

// Post performs an HTTP POST against the indicated endpoint
func (s *DefaultClient) Post(endpoint string, payload io.Reader) ([]byte, *http.Response, error) {
	return s.http(http.MethodPost, endpoint, payload)
}

// Put performs an HTTP PUT against the indicated endpoint
func (s *DefaultClient) Put(endpoint string, payload io.Reader) ([]byte, *http.Response, error) {
	return s.http(http.MethodPut, endpoint, payload)
}

// Del performs an HTTP DELETE against the indicated endpoint
func (s *DefaultClient) Del(endpoint string) (resp *http.Response, err error) {
	_, resp, err = s.http(http.MethodDelete, endpoint, nil)
	return
}

// Info return information about the Nexus server
func (s *DefaultClient) Info() ServerInfo {
	return ServerInfo{s.Host, s.Username, s.Password, s.Repository}
}

// SetDebug will enable or disable debug output on HTTP communication
func (s *DefaultClient) SetDebug(enable bool) {
	s.Debug = enable
}

// SearchQueryBuilder is the interface that a search builder should follow
type SearchQueryBuilder interface {
	Build() string
}

// Status is used as a "ping" of the server. The endpoint returns a non-200
// code when the server is unable to serve requests or the credentials are invalid.
func (s *DefaultClient) TestConn() (err error) {
	_, req, err := s.Get("/")
	if err != nil {
		return
	}
	if req.StatusCode != 200 {
		err = errors.New("Credentials are invalid or Nexus is unable to serve requests")
	}
	return
}
