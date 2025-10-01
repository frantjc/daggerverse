// A generated module for TLS functions

package main

import (
	"fmt"
	"path"
	"time"

	"github.com/frantjc/daggerverse/tls/internal/dagger"
)

type TLS struct {
	// + private
	Container *dagger.Container
}

const (
	workDir   = "/tmp"
	keyPath   = workDir + "/tls.key"
	crtPath   = workDir + "/tls.crt"
	caKeyPath = workDir + "/ca.key"
	caCrtPath = workDir + "/ca.crt"
	days      = "365"
)

var (
	year = fmt.Sprint(time.Now().Year())
)

func New() *TLS {
	return &TLS{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{Packages: []string{"openssl"}}).
			WithExec([]string{
				"openssl", "genrsa",
				"-out", caKeyPath,
				"4096",
			}).
			WithEnvVariable("_TLS_CACHE", year),
	}
}

type TLSCA struct {
	// + private
	Container *dagger.Container
}

func (m *TLS) CA() *TLSCA {
	return &TLSCA{
		Container: m.Container.
			WithExec([]string{
				"openssl", "req", "-new", "-x509",
				"-key", caKeyPath,
				"-out", caCrtPath,
				"-days", days,
				"-subj", "/C=US/ST=State/L=City/O=Dagger CA/OU=CA/CN=Dagger Root CA",
			}),
	}
}

func (m *TLSCA) Crt() *dagger.File {
	return m.Container.File(caCrtPath)
}

type TLSKeyPair struct {
	// + private
	Container *dagger.Container
}

func (m *TLSCA) KeyPair(hostname string) *TLSKeyPair {
	csrPath := fmt.Sprintf("%s/%s.csr", workDir, hostname)
  extPath := fmt.Sprintf("%s/%s.ext", workDir, hostname)

	return &TLSKeyPair{
		Container: m.Container.
			WithExec([]string{
				"openssl", "genrsa",
				"-out", keyPath,
				"4096",
			}).
			WithFile(
				extPath,
				dag.File(path.Base(extPath), fmt.Sprintf("subjectAltName=DNS:%s", hostname)),
			).
			WithExec([]string{
					"openssl", "req", "-new",
					"-key", keyPath,
					"-out", csrPath,
					"-subj", fmt.Sprintf(
							"/C=US/ST=State/L=City/O=Dagger/OU=%s/CN=%s", hostname, hostname),
			}).
			WithExec([]string{
					"openssl", "x509", "-req", "-in", csrPath, "-CA", caCrtPath, "-CAkey", caKeyPath,
					"-CAcreateserial", "-out", crtPath, "-days", days, "-sha256",
					"-extensions", "v3_req", "-extfile", extPath,
			}),
	}
}

func (m *TLSKeyPair) Key() *dagger.File {
	return m.Container.File(keyPath)
}

func (m *TLSKeyPair) Crt() *dagger.File {
	return m.Container.File(crtPath)
}
