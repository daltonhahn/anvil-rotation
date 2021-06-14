package main

import (
        "os"
	//"os/exec"
	"net"
        "strconv"
        crand "crypto/rand"
        "crypto/rsa"
        "crypto/x509"
        "crypto/x509/pkix"
        "encoding/pem"
        "io/ioutil"
        "math/big"
        "time"
        "sync"
	"log"
        "math/rand"
        b64 "encoding/base64"
	"path/filepath"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
  rand.NewSource(time.Now().UnixNano()))

var semaphore = make(chan struct{}, 250)

func StringWithCharset(length int, charset string) string {
  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return b64.StdEncoding.EncodeToString(b)
}

func GenCA(iteration int, numQ int) {
        newpath := filepath.Join(".", "config", strconv.Itoa(iteration))
        os.MkdirAll(newpath, os.ModePerm)
        CAkeyBytes, _ := rsa.GenerateKey(crand.Reader, 4096)
        pemfile, _ := os.Create("config/"+strconv.Itoa(iteration)+"/ca.key")
        var pemkey = &pem.Block{
                Type : "RSA PRIVATE KEY",
                Bytes : x509.MarshalPKCS1PrivateKey(CAkeyBytes)}
        pem.Encode(pemfile, pemkey)
        pemfile.Close()

        CAcert := &x509.Certificate{
                SerialNumber: big.NewInt(2019),
                Subject: pkix.Name{
                        Organization:  []string{"Anvil"},
                        Country:       []string{"US"},
                        Province:      []string{""},
                        Locality:      []string{""},
                        StreetAddress: []string{""},
                        PostalCode:    []string{""},
                },
                NotBefore:              time.Now(),
                NotAfter:               time.Now().AddDate(10, 0, 0),
                IsCA:                   true,
                ExtKeyUsage:            []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
                KeyUsage:               x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
                BasicConstraintsValid:  true,
        }

        certBytes, err := x509.CreateCertificate(crand.Reader, CAcert, CAcert, &CAkeyBytes.PublicKey, CAkeyBytes)
        if err != nil {
                log.Fatalln(err)
        }

        certPEM, _ := os.Create("config/"+strconv.Itoa(iteration)+"/ca.crt")
        pem.Encode(certPEM, &pem.Block{
                Type:  "CERTIFICATE",
                Bytes: certBytes,
	})

	// Make gofunc()
	for i := 1; i < numQ+1; i++ {
		ips, _ := net.LookupIP("server"+strconv.Itoa(i))
		var targIP net.IP
		if ips[0].Equal(net.ParseIP("127.0.0.1")) {
			targIP = ips[1]
		} else {
			targIP = ips[0]
		}

		keyBytes, _ := rsa.GenerateKey(crand.Reader, 4096)
		pemfile, _ := os.Create("config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".key")
		var pemkey = &pem.Block{
			Type : "RSA PRIVATE KEY",
			Bytes : x509.MarshalPKCS1PrivateKey(keyBytes)}
		pem.Encode(pemfile, pemkey)
		pemfile.Close()

		cert := &x509.Certificate{
			SerialNumber: big.NewInt(2019),
			Subject: pkix.Name{
				Organization:  []string{"Anvil"},
				Country:       []string{"US"},
				Province:      []string{""},
				Locality:      []string{""},
				StreetAddress: []string{""},
				PostalCode:    []string{""},
			},
			DNSNames:		[]string{"server"+strconv.Itoa(i)},
			IPAddresses:		[]net.IP{targIP},
			NotBefore:              time.Now(),
			NotAfter:               time.Now().AddDate(10, 0, 0),
			IsCA:                   true,
			ExtKeyUsage:            []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:               x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			BasicConstraintsValid:  true,
		}

	        certBytes, err := x509.CreateCertificate(crand.Reader, cert, CAcert, &keyBytes.PublicKey, CAkeyBytes)
		if err != nil {
			log.Fatalln(err)
		}

		certPEM, _ := os.Create("config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".crt")
		pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		})
	}
}

func GenPairs(nodeName string, iteration int, wg *sync.WaitGroup) {
        semaphore <- struct{}{}
	caPublicKeyFile, err := ioutil.ReadFile("config/"+strconv.Itoa(iteration)+"/ca.crt")
	if err != nil {
		panic(err)
	}
	pemBlock, _ := pem.Decode(caPublicKeyFile)
	if pemBlock == nil {
		panic("pem.Decode failed")
	}
	caCRT, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		panic(err)
	}
	caPrivateKeyFile, err := ioutil.ReadFile("config/"+strconv.Itoa(iteration)+"/ca.key")
	if err != nil {
		panic(err)
	}
	pemBlock, _ = pem.Decode(caPrivateKeyFile)
	if pemBlock == nil {
		panic("pem.Decode failed")
	}
	caPrivateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		panic(err)
	}
        keyBytes, _ := rsa.GenerateKey(crand.Reader, 4096)
        pemfile, _ := os.Create("artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".key")
        var pemkey = &pem.Block{
		Type : "RSA PRIVATE KEY",
		Bytes : x509.MarshalPKCS1PrivateKey(keyBytes)}
	pem.Encode(pemfile, pemkey)
        pemfile.Close()

	ips, _ := net.LookupIP(nodeName)
	var targIP net.IP
	if ips[0].Equal(net.ParseIP("127.0.0.1")) {
		targIP = ips[1]
	} else {
		targIP = ips[0]
	}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Anvil"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		DNSNames:     []string{nodeName},
		IPAddresses:  []net.IP{targIP},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certBytes, err := x509.CreateCertificate(crand.Reader, cert, caCRT, &keyBytes.PublicKey, caPrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	certPEM, _ := os.Create("artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".crt")
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	<-semaphore
	defer wg.Done()
}
