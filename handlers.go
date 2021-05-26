package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	   "crypto/rand"
    "crypto/x509"
    "encoding/pem"
    "io/ioutil"
    "math/big"
    "time"
)

func GenerateUDPKey() {
	fmt.Println("Generating UDP")
}

func GenerateTLSArtifacts(nodeList []string, iteration int) {
	newpath := filepath.Join(".", "artifacts", strconv.Itoa(iteration))
	os.MkdirAll(newpath, os.ModePerm)
	for _, ele := range nodeList {
		nodePath := filepath.Join(newpath, ele)
		os.MkdirAll(nodePath, os.ModePerm)
	}
	fmt.Println("Generating TLS artifacts for ", nodeList)
	genCert()
}

func GenerateACLArtifacts(serviceMap []string) {
	fmt.Println("Generating ACL artifacts for ", serviceMap)
}

func genCert() {
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
    der, err := x509.DecryptPEMBlock(pemBlock, []byte("123456"))
    if err != nil {
        panic(err)
    }
    caPrivateKey, err := x509.ParsePKCS1PrivateKey(der)
    if err != nil {
	fmt.Println("NOT PARSING")
        panic(err)
    }
       // load client certificate request
    clientCSRFile, err := ioutil.ReadFile("config/bob.csr")
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

    certBytes, _ := x509.CreateCertificate(rand.Reader, clientCRTTemplate, caCRT, clientCSR.PublicKey, caPrivateKey)
    clientCRTFile, err := os.Create("config/bob.crt")
    if err != nil {
        panic(err)
    }
    pem.Encode(clientCRTFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
    clientCRTFile.Close()
}

