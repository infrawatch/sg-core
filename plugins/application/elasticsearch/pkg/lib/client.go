package lib

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/google/uuid"
)

// ElasticSearch client implementation using official library from ElasticClient
// TODO: Move this module to infrawatch/apputils

// Client holds cluster connection configuration
type Client struct {
	conn *esv7.Client
}

// NewElasticClient constructor
func NewElasticClient(cfg *AppConfig) (*Client, error) {
	client := &Client{}
	return client, client.Connect(cfg)
}

// createTLSConfig creates appropriate TLS configuration with enabled cert-based authentication
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
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}
	if len(serverName) == 0 {
		tlsConfig.InsecureSkipVerify = true
	} else {
		tlsConfig.ServerName = serverName
	}

	return tlsConfig, nil
}

// Connect initiates connection with ES host and tests the connection
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

	esCfg := esv7.Config{
		Addresses: []string{cfg.HostURL},
		Transport: transport,
	}
	if cfg.UseBasicAuth {
		esCfg.Username = cfg.User
		esCfg.Password = cfg.Password
	}

	esc.conn, err = esv7.NewClient(esCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize connection: %s", err.Error())
	}

	res, err := esc.conn.Info()
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)
	return err
}

// IndicesExists returns true if given indices exists, otherwise return false
func (esc *Client) IndicesExists(indices []string) (bool, error) {
	res, err := esc.conn.Indices.Exists(indices)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("failed to check for indices [%s]: %d", strings.Join(indices, ","), res.StatusCode)
}

// IndicesDelete deletes given indexes. Does not fail if given index does not exist.
func (esc *Client) IndicesDelete(indices []string) error {
	res, err := esc.conn.Indices.Delete(indices)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("failed to delete indices [%d]: %s", res.StatusCode, body)
	}else{
		io.Copy(ioutil.Discard, res.Body)
	}
	return nil
}

// IndicesCreate ...
func (esc *Client) IndicesCreate(indices []string) error {
	for _, index := range indices {
		res, err := esc.conn.Indices.Create(index)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(res.Body)
			msg := string(body)
			if strings.Contains(msg, "resource_already_exists_exception") {
				return nil
			}
			return fmt.Errorf("failed to create index [%d]: %s", res.StatusCode, msg)
		}else{
			io.Copy(ioutil.Discard, res.Body)
		}
	}
	return nil
}

// Index saves given documents under given index
func (esc *Client) Index(index string, documents []string, bulk bool) error {
	if !bulk {
		for _, doc := range documents {
			res, err := esc.conn.Index(index, strings.NewReader(doc))
			if err != nil {
				return err
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
				body, _ := ioutil.ReadAll(res.Body)
				return fmt.Errorf("failed to index document[%d]: %s", res.StatusCode, body)
			}else{
				io.Copy(ioutil.Discard, res.Body)
			}
		}
	} else {
		res, err := esc.conn.Bulk(strings.NewReader(formatBulkRequest(index, documents)))
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("failed to index document(s)[%d]: %s", res.StatusCode, body)
		}else{
			io.Copy(ioutil.Discard, res.Body)
		}
	}
	return nil
}

func generateDocumentID() string {
	id := uuid.New()
	return id.String()
}

func formatBulkRequest(index string, documents []string) string {
	var buffer bytes.Buffer
	for _, doc := range documents {
		buffer.WriteString(fmt.Sprintf("{\"index\":{\"_index\":\"%s\",\"_id\":\"%s\"}}\n", index, generateDocumentID()))
		buffer.WriteString(fmt.Sprintf("%s\n", doc))
	}
	buffer.WriteString("\n")
	return buffer.String()
}
