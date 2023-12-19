package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	pkcs8 "github.com/youmark/pkcs8"
	"golang.org/x/crypto/ed25519"
)

func loadSSLCertificate() tls.Certificate {
	var cert tls.Certificate
	var err error

	// configure ssl
	if *flag_s_ssl_public_key != "" && *flag_s_ssl_private_key != "" {
		cert, err = tls.LoadX509KeyPair(*flag_s_ssl_public_key, *flag_s_ssl_private_key)
		if err != nil {
			if isPEMDecryptorNotFoundError(err) {
				decryptedKey, decryptErr := decryptPrivateKey(*flag_s_ssl_private_key, *flag_s_ssl_private_key_password)
				if decryptErr != nil {
					fatalf_stderr("Failed to decrypt private key: %v", decryptErr)
				}

				var pemBlock *pem.Block

				switch key := decryptedKey.(type) {
				case *rsa.PrivateKey:
					pemBlock = &pem.Block{
						Type:  "RSA PRIVATE KEY",
						Bytes: x509.MarshalPKCS1PrivateKey(key),
					}
				case *ecdsa.PrivateKey:
					ecPrivateKeyBytes, _ := x509.MarshalECPrivateKey(key)
					pemBlock = &pem.Block{
						Type:  "EC PRIVATE KEY",
						Bytes: ecPrivateKeyBytes,
					}
				case ed25519.PrivateKey:
					pemBlock = &pem.Block{
						Type:  "OPENSSH PRIVATE KEY",
						Bytes: key,
					}
				default:
					fatalf_stderr("Unsupported key type: %T", decryptedKey)
				}

				privateKeyPem := pem.EncodeToMemory(pemBlock)

				var errKeyPair error
				cert, errKeyPair = tls.X509KeyPair([]byte(*flag_s_ssl_public_key), privateKeyPem)
				if errKeyPair != nil {
					fatalf_stderr("Failed to load X509 key pair: %v", errKeyPair)
				}
			}
		}
	}

	// configure auto-tls
	if err != nil || *flag_b_auto_ssl {
		privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

		notBefore := time.Now()
		notAfter := notBefore.Add(time.Duration(*flag_i_auto_ssl_default_expires) * time.Hour)
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

		var valid_ips []net.IP
		if strings.Contains(*flag_s_auto_ssl_san_ip, ",") {
			// assume CSV entry of IP addresses
			flag_ips := strings.ReplaceAll(*flag_s_auto_ssl_san_ip, " ", "")
			ips := strings.Split(flag_ips, ",")
			for _, ip := range ips {
				parsed_ip := net.ParseIP(ip)
				if parsed_ip == nil {
					fatalf_stderr("failed to parse the ip address %v", *flag_s_auto_ssl_san_ip)
				}
				valid_ips = append(valid_ips, parsed_ip)
			}
		}

		var valid_domains []string
		valid_domains = append(valid_domains, *flag_s_auto_ssl_domain_name)
		if strings.Contains(*flag_s_auto_ssl_additional_domains, ",") {
			flag_domains := strings.ReplaceAll(*flag_s_auto_ssl_additional_domains, " ", "")
			domains := strings.Split(flag_domains, ",")
			for _, domain := range domains {
				_, err := url.Parse(domain)
				if err != nil {
					log.Printf("Invalid domain: %s\n", domain)
				} else {
					valid_domains = append(valid_domains, domain)
				}
			}
		}

		template := x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Organization: []string{*flag_s_auto_ssl_company},
				CommonName:   *flag_s_auto_ssl_domain_name,
			},
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			IPAddresses:           valid_ips,
			DNSNames:              valid_domains,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)

		cert = tls.Certificate{
			Certificate: [][]byte{derBytes},
			PrivateKey:  privateKey,
		}
	}

	return cert
}

func startCertReloader() {
	ticker := time.NewTicker(time.Duration(*flag_i_reload_cert_every_minutes) * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				newCert := loadSSLCertificate()
				mu_cert.Lock()
				cert = newCert
				mu_cert.Unlock()
			case <-ch_cert_reloader_cancel:
				ticker.Stop()
				return
			}
		}
	}()
}

func getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	mu_cert.RLock()
	defer mu_cert.RUnlock()
	return &cert, nil
}

func decryptPrivateKey(privateKeyPath, password string) (crypto.PrivateKey, error) {
	keyFile, err := os.Open(privateKeyPath)
	if err != nil {
		return nil, err
	}
	defer keyFile.Close()

	var keyBytes []byte
	buf := make([]byte, 1024)
	for {
		n, err := keyFile.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		keyBytes = append(keyBytes, buf[:n]...)
	}

	pemBlock, _ := pem.Decode(keyBytes)
	if pemBlock == nil {
		return nil, errors.New("could not decode PEM block of private key")
	}

	if strings.Contains(string(keyBytes), "ENCRYPTED") {
		privKey, err := pkcs8.ParsePKCS8PrivateKey(pemBlock.Bytes, []byte(password))
		if err != nil {
			return nil, err
		}
		return privKey, nil
	} else {
		privKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		return privKey, nil
	}
}

func isPEMDecryptorNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "x509: decryption password incorrect")
}
