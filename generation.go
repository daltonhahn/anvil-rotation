package main

import (
        "os"
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
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
  rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return b64.StdEncoding.EncodeToString(b)
}

func GenPairs(nodeName string, iteration int, wg *sync.WaitGroup) {
        semaphore <- struct{}{}
	caPublicKeyFile, err := ioutil.ReadFile("config/ca.crt")
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
	caPrivateKeyFile, err := ioutil.ReadFile("config/ca.key")
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
        keyBytes, _ := rsa.GenerateKey(crand.Reader, 4096)
        pemfile, _ := os.Create("artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".key")
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
		DNSNames:     []string{nodeName},
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


/*
func genCert(nodeName string, iteration int, wg *sync.WaitGroup) {
        semaphore <- struct{}{}
    // load CA key pair
    //      public key
    caPublicKeyFile, err := ioutil.ReadFile("config/ca.crt")
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

    //      private key
    caPrivateKeyFile, err := ioutil.ReadFile("config/ca.key")
    if err != nil {
        panic(err)
    }
    pemBlock, _ = pem.Decode(caPrivateKeyFile)
    if pemBlock == nil {
        panic("pem.Decode failed")
    }
    /*
    der, err := x509.DecryptPEMBlock(pemBlock, []byte(""))
    if err != nil {
        panic(err)
    }
    caPrivateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)//der
    if err != nil {
        fmt.Println("NOT PARSING")
        panic(err)
    }
       // load client certificate request
    fileName := "artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".csr"
    clientCSRFile, err := ioutil.ReadFile(fileName)
    if err != nil {
        panic(err)
    }
    pemBlock, _ = pem.Decode(clientCSRFile)
    if pemBlock == nil {
        panic("pem.Decode failed")
    }
    clientCSR, err := x509.ParseCertificateRequest(pemBlock.Bytes)
    if err != nil {
        panic(err)
    }
    if err = clientCSR.CheckSignature(); err != nil {
        panic(err)
    }

    // create client certificate template
    clientCRTTemplate := &x509.Certificate{
        Signature:          clientCSR.Signature,
        SignatureAlgorithm: clientCSR.SignatureAlgorithm,

        PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
        PublicKey:          clientCSR.PublicKey,

        SerialNumber: big.NewInt(2),
        Issuer:       caCRT.Subject,
        Subject:      clientCSR.Subject,
        NotBefore:    time.Now(),
        NotAfter:     time.Now().Add(24 * time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }

    certBytes, _ := x509.CreateCertificate(crand.Reader, clientCRTTemplate, caCRT, clientCSR.PublicKey, caPrivateKey)
    certName := "artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".crt"
    clientCRTFile, err := os.Create(certName)
    if err != nil {
        panic(err)
    }
    pem.Encode(clientCRTFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
    clientCRTFile.Close()
    <-semaphore
    defer wg.Done()
}
*/
