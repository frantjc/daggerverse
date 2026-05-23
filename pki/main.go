package main

import (
	"fmt"

	"dagger.io/dagger"
	"github.com/logsquaredn/rubber/.dagger/modules/pki/internal/dagger"
)

const (
	workDir   = "/tmp"
	caKeyPath = workDir + "/ca.key"
	caCrtPath = workDir + "/ca.crt"
	keyPath   = workDir + "/tls.key"
	crtPath   = workDir + "/tls.crt"
	cfgPath   = workDir + "/cs.cfg"
	pfxPath   = workDir + "/cs.pfx"
	passPath  = workDir + "/pass.txt"
)

// PKI provides a certificate authority and tools to issue TLS and code-signing credentials.
type PKI struct {
	// +private
	Container *dagger.Container
}

func New(
	// An existing CA private key. If omitted, a new RSA-4096 key is generated.
	// +optional
	caKey *dagger.Secret,
	// +optional
	container *dagger.Container,
) *PKI {
	if container == nil {
		container = dag.Wolfi().
			Container(dagger.WolfiContainerOpts{Packages: []string{"openssl"}})
	}

	if caKey != nil {
		return &PKI{
			Container: container.WithMountedSecret(caKeyPath, caKey),
		}
	}

	return &PKI{
		Container: container.WithExec([]string{
			"openssl", "genrsa", "-out", caKeyPath, "4096",
		}),
	}
}

// Key returns the CA private key.
func (m *PKI) Key() *dagger.File {
	return m.Container.File(caKeyPath)
}

// PKICA represents a certificate authority backed by a key and certificate.
type PKICA struct {
	// +private
	Container *dagger.Container
}

// CA initialises or loads a certificate authority.
func (m *PKI) CA(
	// Distinguished name for a newly-generated CA certificate, e.g. "/CN=My CA/O=Acme".
	// Ignored when caCrt is provided.
	// +optional
	// +default="/CN=Root CA"
	subject string,
	// Validity period in days for a newly-generated CA certificate.
	// Ignored when caCrt is provided.
	// +optional
	// +default=3650
	days int,
	// An existing CA certificate. If provided, it is used as-is and subject/days are ignored.
	// +optional
	caCrt *dagger.Secret,
) *PKICA {
	if caCrt != nil {
		return &PKICA{Container: m.Container.WithMountedSecret(caCrtPath, caCrt)}
	} else if days <= 0 {
		days = 3650
	}

	return &PKICA{
		Container: m.Container.WithExec([]string{
			"openssl", "req", "-new", "-x509",
			"-key", caKeyPath,
			"-out", caCrtPath,
			"-days", fmt.Sprint(days),
			"-sha256",
			"-subj", subject,
		}),
	}
}

func (m *PKICA) Crt() *dagger.File {
	return m.Container.File(caCrtPath)
}

// PKITLSKeyPair holds a TLS private key and certificate signed by the CA.
type PKITLSKeyPair struct {
	// +private
	Container *dagger.Container
}

// TLSKeyPair issues a TLS server certificate for the given hostname, signed by the CA.
func (m *PKICA) TLSKeyPair(
	// Hostname used as the certificate CN and DNS SAN.
	hostname string,
	// Validity period in days.
	// +optional
	// +default=365
	days int,
) *PKITLSKeyPair {
	if days <= 0 {
		days = 365
	}

	csrPath := fmt.Sprintf("%s/%s.csr", workDir, hostname)
	extPath := fmt.Sprintf("%s/%s.ext", workDir, hostname)

	return &PKITLSKeyPair{
		Container: m.Container.
			WithExec([]string{"openssl", "genrsa", "-out", keyPath, "4096"}).
			WithNewFile(extPath, fmt.Sprintf(`[v3_req]
subjectAltName=DNS:%s
extendedKeyUsage=serverAuth
`,
				hostname,
			)).
			WithExec([]string{
				"openssl", "req", "-new",
				"-key", keyPath,
				"-out", csrPath,
				"-subj", fmt.Sprintf("/CN=%s", hostname),
			}).
			WithExec([]string{
				"openssl", "x509", "-req",
				"-in", csrPath,
				"-CAkey", caKeyPath,
				"-CA", caCrtPath,
				"-CAcreateserial",
				"-out", crtPath,
				"-days", fmt.Sprint(days),
				"-sha256",
				"-extensions", "v3_req",
				"-extfile", extPath,
			}),
	}
}

// Key returns the TLS private key.
func (m *PKITLSKeyPair) Key() *dagger.File {
	return m.Container.File(keyPath)
}

// Crt returns the TLS certificate.
func (m *PKITLSKeyPair) Crt() *dagger.File {
	return m.Container.File(crtPath)
}

type PKICodesignKeyPair struct {
	// +private
	Container *dagger.Container
	// +private
	Name string
}

// CodesignKeyPair issues a code-signing certificate signed by the CA and packages it as a PKCS#12 bundle.
func (m *PKICA) CodesignKeyPair(
	// Friendly name used as both the certificate CN and the PKCS#12 bag label.
	name string,
	// Password protecting the PKCS#12 bundle.
	password *dagger.Secret,
	// Validity period in days.
	// +optional
	// +default=365
	days int,
	// Full distinguished name for the signing cert, e.g. "/CN=Acme/O=Acme Inc/OU=Dev".
	// Defaults to "/CN=<name>" when empty.
	// +optional
	subject string,
) *PKICodesignKeyPair {
	if days <= 0 {
		days = 365
	}

	if subject == "" {
		subject = fmt.Sprintf("/CN=%s", name)
	}

	return &PKICodesignKeyPair{
		Container: m.Container.
			WithNewFile(cfgPath, `[ext]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature
extendedKeyUsage = codeSigning
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer
`).
			WithExec([]string{
				"openssl", "req", "-x509",
				"-newkey", "rsa:4096",
				"-sha256",
				"-nodes",
				"-days", fmt.Sprint(days),
				"-CAkey", caKeyPath,
				"-CA", caCrtPath,
				"-subj", subject,
				"-config", cfgPath,
				"-keyout", keyPath,
				"-out", crtPath,
			}).
			WithMountedSecret(passPath, password).
			WithExec([]string{
				"openssl", "pkcs12", "-export",
				"-keypbe", "AES-256-CBC",
				"-certpbe", "AES-256-CBC",
				"-macalg", "SHA256",
				"-inkey", keyPath,
				"-in", crtPath,
				"-name", name,
				"-passout", fmt.Sprintf("file:%s", passPath),
				"-out", pfxPath,
			}),
	}
}

// Pfx returns the PKCS#12 bundle.
func (m *PKICodesignKeyPair) Pfx() *dagger.File {
	return m.Container.File(pfxPath)
}
