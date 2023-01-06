package lib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testESURL   = "http://localhost:9200"
	testESIndex = "unit-test"
)

var (
	testCert = `-----BEGIN CERTIFICATE-----
MIIECTCCAvGgAwIBAgIUFfcg+wm/K2XnER/xSRoZb3kRc+8wDQYJKoZIhvcNAQEL
BQAwgZMxCzAJBgNVBAYTAkNaMRAwDgYDVQQIDAdCcmVjbGF2MRAwDgYDVQQHDAdN
aWt1bG92MRAwDgYDVQQKDAdSZWQgSGF0MREwDwYDVQQLDAhDbG91ZE9wczEfMB0G
A1UEAwwWd3ViYmFsdWJiYS5sb2NhbGRvbWFpbjEaMBgGCSqGSIb3DQEJARYLc2dA
Y29yZS5jb20wHhcNMjEwMzE4MTIwNTA3WhcNMjEwNDE3MTIwNTA3WjCBkzELMAkG
A1UEBhMCQ1oxEDAOBgNVBAgMB0JyZWNsYXYxEDAOBgNVBAcMB01pa3Vsb3YxEDAO
BgNVBAoMB1JlZCBIYXQxETAPBgNVBAsMCENsb3VkT3BzMR8wHQYDVQQDDBZ3dWJi
YWx1YmJhLmxvY2FsZG9tYWluMRowGAYJKoZIhvcNAQkBFgtzZ0Bjb3JlLmNvbTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM2t0U5duzAQ+6wz1c9gp1rA
DM2UaOdgRYIWv8rQkBe5bstat5nAJO0A+KcqxxdMOjyN0JWZgYjhBP57lq/RMoIE
sT4UEFm5QjeMmBb0PM7t6KSbGlRNKgMr3cRku7yPatZijZXEpfWxcQ+0BgOPY/Hr
pKRByDHhGtZi3vBeFE6LtxBI6Fmp9BS/hoODqEDaVIsVa7NsmWXIsoV99mSZ6k2u
9yZHijk9wpmMdq1BL9zGSTBiOyALLhOrOyuissJh6Lpcd2AWcxpmjyTnia8xV8eV
g1o2OfL/y0Yo8wwW+gR1ij1bOevJ+b7Oc07p/3mqc+0+JSYbw1g11CPOzKx4WSsC
AwEAAaNTMFEwHQYDVR0OBBYEFM3awe2EvTVuFq+4ssHVqIFuvzoOMB8GA1UdIwQY
MBaAFM3awe2EvTVuFq+4ssHVqIFuvzoOMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZI
hvcNAQELBQADggEBAFnmpkd88ym/sgQ6S2RvyQtTt171PZUDebyJAKfWpgdBJXuv
ztvI8glNvle2+6wQToc5GbPGVqy5jDG7HKrexBwE6Fp8omZEnRJLc7C4yOHZQVfm
MHGBcs6JmJPn7T+69Qz/sxbINXqOVz0LDrpS/MS7n7ioXiI5KlZFTnhgLpLPqdjq
2U4cxqygFo4fYeLvxTGpk3IAPc4h7Yrna11z1ribw5vsODILGxL0ENSbSz2dopLm
yHpzk3RjAZaVLRifW+qsLI/TRxa0lEnEdohbIVbQxr3y3VOPEzJvz/PyLFdFC3uf
4VjFRzMHHwi2SFjBWO3105jm9gor13df0yjbdVE=
-----END CERTIFICATE-----
`
	testKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAza3RTl27MBD7rDPVz2CnWsAMzZRo52BFgha/ytCQF7luy1q3
mcAk7QD4pyrHF0w6PI3QlZmBiOEE/nuWr9EyggSxPhQQWblCN4yYFvQ8zu3opJsa
VE0qAyvdxGS7vI9q1mKNlcSl9bFxD7QGA49j8eukpEHIMeEa1mLe8F4UTou3EEjo
Wan0FL+Gg4OoQNpUixVrs2yZZciyhX32ZJnqTa73JkeKOT3CmYx2rUEv3MZJMGI7
IAsuE6s7K6KywmHoulx3YBZzGmaPJOeJrzFXx5WDWjY58v/LRijzDBb6BHWKPVs5
68n5vs5zTun/eapz7T4lJhvDWDXUI87MrHhZKwIDAQABAoIBAEqeJLEpkB+ACc4P
gCIcDpr90adDkEtgwdbQKgSKZbw1qdxcrP86lirlj1AWVOQ+42HUkTe02SmveQBa
FfDzFD/XM/YxkTz72OoON58cPHNWHHCbVJIA7Jz57Rqy8OkXnsroNjV/gjYAieQI
i6X+/2Nk+fYdZ2OxJutgM0FA4F0d7DsFKX83GYkrfwakqhBMN6A1Po0KYGJIZ/wd
l4z336U7Kol78Kt9xYgHBKtWReM6qjG6xgw9FUBRzis+dcyYEDxSup3Y7u2N9hvd
oSsvaApKo46DQOQJiXGGJ3wuPqOJjIP+qYVwxPZaPEpRXvTJBuuepZE6YUwBxoLo
QQlsXAECgYEA8N1bqtBAd/EjzGkFlV/vPRBmT2VZt3wdiPONBkywIZEs95oObE/l
3ZXfKkMPCI67N6JJiLHFJ4oueR6Usaz0zPUcTlkioO5GfFWvqXegFnmpK2Ae6mhI
zOU4j+Uq8HCijNILXyNf6qgulfIPLFkhVERfNHSfv5TU0gQIiPlQBFECgYEA2ppy
4HYcYkuFlmtr+jCdI2eYYjlOABjG9dbKey10KDUn5rCs8sNJlbSj2+To7HT4qbZa
8F/EzbbkT7FExrxiHfZ9PjXx9e+EpHKqpIPg+8X4QPdC5xCh1L3MoSVPUsZifnqy
QAlAQakRBUgI5Po0aSu9VXmrBLpjOIMwsArfkrsCgYAiN2/0Pg1KfKkXOrweUjiM
Ni4yjTVHiYwwjli0UmSbACKhMfNmk5sV9Vp0iH40OwKBjr5fetGFIm4jqqJ48xb7
nr5cqvDuZ6r/srR3oJTPXI0Zqlf5+MKOyOlWF7oX2ghddOFErKPNlAK6Ll7Vb/v6
GpRjwUWIU74/726+9pvVYQKBgQCgjRDUBEsicj8h07GRJgUzDJHhih7ceVYfFmrN
/vsx0KCGkLnk7kLsHai/Bqd/iwVad+DgbCX5xFp4oURXBeK2COPBPhOAQjLUKJdl
jqo9oA+Nf0x2skN5IRDaRbG1pJiQNgMWfvTfhJFIpLhLm+vEVmiPD3XoWhAnYErw
8Ht1owKBgE/WCqAjsD1t6cmNX2UDqsXQULQYuWHEftp3lzHRPClF6zAnwSGUDNrf
7f5lcspAI9sun4veId/ox9eGbrz9ldw46grPy/Zkflk2eFTUStME1Pt8GAXEF/pb
/5f2IAPbX29zG1UGfS5YmNLLLQYZl/oFaR3ZKNAyvklectylEjmM
-----END RSA PRIVATE KEY-----
`
	testCa = `-----BEGIN CERTIFICATE-----
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
/7AvggsgSpAMLS6QwA==
-----END CERTIFICATE-----
`
)

func TestElasticsearchTLSConf(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "connector_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	certpath := path.Join(tmpdir, "cert.pem")
	require.NoError(t, os.WriteFile(certpath, []byte(testCert), 0600))
	keypath := path.Join(tmpdir, "key.pem")
	require.NoError(t, os.WriteFile(keypath, []byte(testKey), 0600))
	cacrtpath := path.Join(tmpdir, "ca.pem")
	require.NoError(t, os.WriteFile(cacrtpath, []byte(testCa), 0600))

	t.Run("Test insecure connection.", func(t *testing.T) {
		tlsConf, err := createTLSConfig("overmind.localdomain", certpath, keypath, cacrtpath)
		require.NoError(t, err)
		assert.Equal(t, "overmind.localdomain", tlsConf.ServerName)
		assert.Equal(t, false, tlsConf.InsecureSkipVerify)

		tlsConf, err = createTLSConfig("", certpath, keypath, cacrtpath)
		require.NoError(t, err)
		assert.Equal(t, true, tlsConf.InsecureSkipVerify)
	})
}

func curl(t *testing.T, url string, query string) []byte {
	q := strings.NewReader(query)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, q)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return body
}

func TestElasticsearchClient(t *testing.T) {
	cfg := &AppConfig{
		HostURL: testESURL,
		UseTLS:  false,
	}
	client, err := NewElasticClient(cfg)
	require.NoError(t, err)

	t.Run("Test non-existent index delete", func(t *testing.T) {
		resp := curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		assert.Equal(t, false, strings.Contains(string(resp), testESIndex))

		exists, err := client.IndicesExists([]string{testESIndex})
		require.NoError(t, err)
		assert.Equal(t, false, exists)

		err = client.IndicesDelete([]string{testESIndex})
		require.NoError(t, err)

		resp = curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		assert.Equal(t, false, strings.Contains(string(resp), testESIndex))
	})

	t.Run("Test indexing: single document, no bulk", func(t *testing.T) {
		err := client.Index(testESIndex, []string{"{\"id\":1,\"name\":\"single-document-test\"}"}, false)
		require.NoError(t, err)
		time.Sleep(time.Second)
		// verify index creation
		resp := curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		assert.Equal(t, true, strings.Contains(string(resp), testESIndex))
		// verify document existence
		resp = curl(t, fmt.Sprintf("%s/%s/_search", testESURL, testESIndex), "{\"query\":{\"match_all\":{}}}")
		assert.Equal(t, true, strings.Contains(string(resp), "single-document-test"))
	})

	t.Run("Test indexing: multiple documents, bulk mode", func(t *testing.T) {
		records := []string{
			"{\"id\":2,\"name\":\"multi-document-test-1\"}",
			"{\"id\":3,\"name\":\"multi-document-test-2\"}",
			"{\"id\":4,\"name\":\"multi-document-test-3\"}",
		}
		err := client.Index(testESIndex, records, true)
		require.NoError(t, err)
		time.Sleep(time.Second)
		// verify index creation
		resp := curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		assert.Equal(t, true, strings.Contains(string(resp), testESIndex))
		// verify document existence
		resp = curl(t, fmt.Sprintf("%s/%s/_search", testESURL, testESIndex), "{\"query\":{\"match_all\":{}}}")
		assert.Equal(t, true, strings.Contains(string(resp), "multi-document-test-1"))
		assert.Equal(t, true, strings.Contains(string(resp), "multi-document-test-2"))
		assert.Equal(t, true, strings.Contains(string(resp), "multi-document-test-3"))
	})

	t.Run("Test index existence check and delete", func(t *testing.T) {
		resp := curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		exists, err := client.IndicesExists([]string{testESIndex})
		require.NoError(t, err)
		assert.Equal(t, exists, strings.Contains(string(resp), testESIndex))
		assert.Equal(t, true, exists)

		err = client.IndicesDelete([]string{testESIndex})
		require.NoError(t, err)

		resp = curl(t, fmt.Sprintf("%s/_aliases", testESURL), "")
		assert.Equal(t, false, strings.Contains(string(resp), testESIndex))
	})

	t.Run("Test index initialization", func(t *testing.T) {
		indices := []string{"test1", "test2"}
		// cleanup
		err = client.IndicesDelete(indices)
		require.NoError(t, err)
		// try indices creation
		err := client.IndicesCreate(indices)
		require.NoError(t, err)
		// verify existence
		exists, err := client.IndicesExists(indices)
		require.NoError(t, err)
		assert.Equal(t, true, exists)
		// try recreation of existent indices
		err = client.IndicesCreate(indices)
		require.NoError(t, err)
	})
}
