package tlshelpers

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/onsi/gomega"
)

func GenerateCa() (string, *rsa.PrivateKey) {
	caFileName, privKey, err := buildCaFile()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return caFileName, privKey
}

func GenerateCertAndKey(caFileName string, caPrivateKey *rsa.PrivateKey) (clientCertFileName string, clientPrivateKeyFileName string, cert tls.Certificate) {
	certPem, keyPem, err := buildCertPem(caPrivateKey, caFileName)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	clientCertFileName = writeClientCredFile(certPem)
	clientPrivateKeyFileName = writeClientCredFile(keyPem)

	cert, err = tls.X509KeyPair(certPem, keyPem)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return
}

func GenerateCaAndMutualTlsCerts() (caFileName string, certFileName string, privateKeyFileName string, cert tls.Certificate) {
	caFileName, privKey := GenerateCa()
	certFileName, privateKeyFileName, cert = GenerateCertAndKey(caFileName, privKey)
	return
}

func CertPool(certName string) *x509.CertPool {
	certPool := x509.NewCertPool()
	caCertificate := mapToX509Cert(certName)
	gomega.Expect(caCertificate).To(gomega.HaveLen(1))
	certPool.AddCert(caCertificate[0])
	return certPool
}

func mapToX509Cert(PemEncodedCertFilePath string) []*x509.Certificate {
	caFileContents, err := os.ReadFile(PemEncodedCertFilePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	caFileBlock, _ := pem.Decode(caFileContents)
	gomega.Expect(caFileBlock).NotTo(gomega.BeNil(), "failed to decode PEM block from %s", PemEncodedCertFilePath)
	caCertificate, err := x509.ParseCertificates(caFileBlock.Bytes)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return caCertificate
}

func buildCaFile() (string, *rsa.PrivateKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", nil, err
	}

	now := time.Now()
	notAfter := now.Add(365 * 24 * time.Hour)

	ca := &x509.Certificate{
		SerialNumber:       serialNumber,
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject: pkix.Name{
			Country:      []string{"USA"},
			Organization: []string{"Cloud Foundry"},
			CommonName:   "CA",
		},
		NotBefore:             now,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,

		IsCA:     true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, ca, ca, &privKey.PublicKey, privKey)
	if err != nil {
		return "", nil, err
	}

	file, err := os.CreateTemp(os.TempDir(), "routing-api-ca")
	if err != nil {
		return "", nil, err
	}

	err = pem.Encode(file, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	})
	if err != nil {
		return "", nil, err
	}

	if err := file.Close(); err != nil {
		return "", nil, err
	}

	return file.Name(), privKey, nil
}

func buildCertPem(privKey *rsa.PrivateKey, caFilePath string) (cert []byte, key []byte, err error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	notAfter := now.Add(365 * 24 * time.Hour)

	serverCertTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"USA"},
			Organization: []string{"Cloud Foundry"},
		},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:             now,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,

		IsCA:               false,
		SignatureAlgorithm: x509.SHA256WithRSA,
		KeyUsage:           x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	caBytes, err := os.ReadFile(caFilePath)
	if err != nil {
		return nil, nil, err
	}

	caBlockDer, _ := pem.Decode(caBytes)
	if caBlockDer == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block from CA file %s", caFilePath)
	}
	caCert, err := x509.ParseCertificate(caBlockDer.Bytes)
	if err != nil {
		return nil, nil, err
	}

	serverCert, err := x509.CreateCertificate(rand.Reader, serverCertTemplate, caCert, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, err
	}

	certBuffer := &bytes.Buffer{}

	err = pem.Encode(certBuffer, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCert,
	})
	if err != nil {
		return nil, nil, err
	}

	keyBuffer := &bytes.Buffer{}

	derEncodedPrivateKey := x509.MarshalPKCS1PrivateKey(privKey)
	err = pem.Encode(keyBuffer, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derEncodedPrivateKey,
	})
	if err != nil {
		return nil, nil, err
	}

	return certBuffer.Bytes(), keyBuffer.Bytes(), nil
}

func writeClientCredFile(data []byte) string {
	tempFile, err := os.CreateTemp(os.TempDir(), "clientcredstest")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(tempFile.Name(), data, 0600)).To(gomega.Succeed())
	return tempFile.Name()
}
