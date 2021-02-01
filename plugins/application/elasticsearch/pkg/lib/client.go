package lib

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

// ElasticSearch client implementation using official library from ElasticClient

//Client holds cluster connection configuration
type Client struct {
	conn *esv7.Client
}

//NewElasticClient constructor
func NewElasticClient(cfg *AppConfig) (*Client, error) {
	client := &Client{}
	return client, client.Connect(cfg)
}

//createTLSConfig creates appropriate TLS configuration with enabled cert-based authentication
func createTLSConfig(serverName string, certFile string, keyFile string, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(ca)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}
	if len(serverName) == 0 {
		tlsConfig.InsecureSkipVerify = true
	} else {
		tlsConfig.ServerName = serverName
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig, nil
}

//Connect initiates connection with ES host and tests the connection
func (esc *Client) Connect(cfg *AppConfig) error {
	var err error

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.UseTLS {
		tlsConfig, err := createTLSConfig(cfg.TLSServerName, cfg.TLSClientCert, cfg.TLSClientKey, cfg.TLSCaCert)
		if err != nil {
			return err
		}
		transport.TLSClientConfig = tlsConfig
	}
	esc.conn, err = esv7.NewClient(esv7.Config{
		Addresses: []string{cfg.HostURL},
		Transport: transport,
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize connection")
	}

	_, err = esc.conn.Info()
	return err
}

//IndicesExists returns true if given indices exists, otherwise return false
func (esc *Client) IndicesExists(indices []string) (bool, error) {
	res, err := esc.conn.Indices.Exists(indices)
	if err != nil {
		return false, err
	}
	if res.StatusCode == 200 {
		return true, nil
	}
	return false, nil
}

//IndicesDelete ...
func (esc *Client) IndicesDelete(indices []string) error {
	res, err := esc.conn.Indices.Delete(indices)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("Failed to delete indices [%d]: %s", res.StatusCode, body)
	}
	return nil
}

//IndicesCreate ...
func (esc *Client) IndicesCreate(indices []string) error {
	for _, index := range indices {
		res, err := esc.conn.Indices.Create(index)
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("Failed to create index [%d]: %s", res.StatusCode, body)
		}
	}
	return nil
}

//Index saves given documents under given index
func (esc *Client) Index(index string, documents []string) error {
	for _, doc := range documents {
		res, err := esc.conn.Index(index, strings.NewReader(doc))
		if err != nil {
			return err
		}
		if res.StatusCode != 200 && res.StatusCode != 201 {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("Failed to index document[%d]: %s", res.StatusCode, body)
		}
	}
	return nil
}
