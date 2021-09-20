package main

import (
        "os"
        "strconv"
        crand "crypto/rand"
        //"crypto/rsa"
	"crypto/ecdsa"
	"crypto/elliptic"
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
        newpath := filepath.Join("/home/anvil/Desktop/anvil-rotation/", "config", strconv.Itoa(iteration))
        os.MkdirAll(newpath, os.ModePerm)
        CAkeyBytes, _ := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
        pemfile, _ := os.Create("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/ca.key")
	marshKeyBytes, _ := x509.MarshalPKCS8PrivateKey(CAkeyBytes)
        var pemkey = &pem.Block{
                Type : "EC PRIVATE KEY",
                Bytes : marshKeyBytes}
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

        certPEM, _ := os.Create("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/ca.crt")
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

			keyBytes, _ := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
			pemfile, _ := os.Create("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/anvilserver"+strconv.Itoa(i)+".key")
			marshKeyBytes, _ := x509.MarshalPKCS8PrivateKey(keyBytes)
			var pemkey = &pem.Block{
				Type : "EC PRIVATE KEY",
				Bytes : marshKeyBytes}
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
				DNSNames:		[]string{"anvilserver"+strconv.Itoa(i)},
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

			certPEM, _ := os.Create("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/anvilserver"+strconv.Itoa(i)+".crt")
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
	caPublicKeyFile, err := ioutil.ReadFile("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/"+prefix+".crt")
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
	caPrivateKeyFile, err := ioutil.ReadFile("/home/anvil/Desktop/anvil-rotation/config/"+strconv.Itoa(iteration)+"/"+prefix+".key")
	if err != nil {
		panic(err)
	}
	pemBlock, _ = pem.Decode(caPrivateKeyFile)
	if pemBlock == nil {
		panic("pem.Decode failed")
	}
	caPrivateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		panic(err)
	}
        keyBytes, _ := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
        pemfile, _ := os.Create("/home/anvil/Desktop/anvil-rotation/artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".key")
	marshKeyBytes, _ := x509.MarshalPKCS8PrivateKey(keyBytes)
        var pemkey = &pem.Block{
		Type : "EC PRIVATE KEY",
		Bytes : marshKeyBytes}
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
		DNSNames:     []string{nodeName},
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

	certPEM, _ := os.Create("/home/anvil/Desktop/anvil-rotation/artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".crt")
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	defer certPEM.Close()

	time.Sleep(2*time.Second)

	<-semaphore
}
