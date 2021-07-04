package main

import (
        "os"
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
        newpath := filepath.Join("/root/anvil-rotation/", "config", strconv.Itoa(iteration))
        os.MkdirAll(newpath, os.ModePerm)
        CAkeyBytes, _ := rsa.GenerateKey(crand.Reader, 2048)
        pemfile, _ := os.Create("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/ca.key")
        var pemkey = &pem.Block{
                Type : "RSA PRIVATE KEY",
                Bytes : x509.MarshalPKCS1PrivateKey(CAkeyBytes)}
        pem.Encode(pemfile, pemkey)
        defer pemfile.Close()

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
		MaxPathLen:             2,
                ExtKeyUsage:            []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
                KeyUsage:               x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
                BasicConstraintsValid:  true,
        }

        certBytes, err := x509.CreateCertificate(crand.Reader, CAcert, CAcert, &CAkeyBytes.PublicKey, CAkeyBytes)
        if err != nil {
                log.Fatalln(err)
        }

        certPEM, _ := os.Create("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/ca.crt")
        pem.Encode(certPEM, &pem.Block{
                Type:  "CERTIFICATE",
                Bytes: certBytes,
	})
	defer certPEM.Close()

	// Make gofunc()
	var wg sync.WaitGroup
	wg.Add(numQ)
	for i := 1; i < numQ+1; i++ {
		go func(i int) {
			ips, err := net.LookupIP("server"+strconv.Itoa(i))
			var targIP net.IP
			if ips[0].Equal(net.ParseIP("127.0.0.1")) {
				targIP = ips[1]
			} else {
				targIP = ips[0]
			}

			keyBytes, _ := rsa.GenerateKey(crand.Reader, 2048)
			pemfile, _ := os.Create("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".key")
			var pemkey = &pem.Block{
				Type : "RSA PRIVATE KEY",
				Bytes : x509.MarshalPKCS1PrivateKey(keyBytes)}
			pem.Encode(pemfile, pemkey)
			defer pemfile.Close()

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
				MaxPathLenZero:         false,
				MaxPathLen:             1,
				ExtKeyUsage:            []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
				KeyUsage:               x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
				BasicConstraintsValid:  true,
			}

			certBytes, err := x509.CreateCertificate(crand.Reader, cert, CAcert, &keyBytes.PublicKey, CAkeyBytes)
			if err != nil {
				log.Fatalln(err)
			}

			certPEM, _ := os.Create("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".crt")
			pem.Encode(certPEM, &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certBytes,
			})
			defer certPEM.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func GenPairs(nodeName string, iteration int, prefix string, quorumMems []string) {
        semaphore <- struct{}{}
	caPublicKeyFile, err := ioutil.ReadFile("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/"+prefix+".crt")
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
	caPrivateKeyFile, err := ioutil.ReadFile("/root/anvil-rotation/config/"+strconv.Itoa(iteration)+"/"+prefix+".key")
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
        keyBytes, _ := rsa.GenerateKey(crand.Reader, 2048)
        pemfile, _ := os.Create("/root/anvil-rotation/artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".key")
        var pemkey = &pem.Block{
		Type : "RSA PRIVATE KEY",
		Bytes : x509.MarshalPKCS1PrivateKey(keyBytes)}
	pem.Encode(pemfile, pemkey)
        defer pemfile.Close()

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
		IsCA:	      false,
		MaxPathLenZero: true,
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certBytes, err := x509.CreateCertificate(crand.Reader, cert, caCRT, &keyBytes.PublicKey, caPrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	certPEM, _ := os.Create("/root/anvil-rotation/artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".crt")
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	defer certPEM.Close()

	time.Sleep(2*time.Second)

	<-semaphore
}
