package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	   "crypto/rand"
	   "crypto/rsa"
    "crypto/x509"
        "crypto/x509/pkix"
    "encoding/asn1"
    "encoding/pem"
    "io/ioutil"
    "math/big"
    "time"
    "sync"
)

var semaphore = make(chan struct{}, 250)

type ACLMap struct {
	TokName		string
	Node            string
        Svc             string
        ValidList       []string
}

func GenerateUDPKey() {
	fmt.Println("Generating UDP")
}

func GenerateTLSArtifacts(nodeList []string, iteration int) {
	start := time.Now()
	var wg sync.WaitGroup
	newpath := filepath.Join(".", "artifacts", strconv.Itoa(iteration))
	os.MkdirAll(newpath, os.ModePerm)
	for _, ele := range nodeList {
		nodePath := filepath.Join(newpath, ele)
		os.MkdirAll(nodePath, os.ModePerm)
	}
	csrStart := time.Now()
	for _, ele := range nodeList {
		wg.Add(1)
		go CSR(ele, iteration, &wg)
	}
	wg.Wait()
	csrDuration := time.Since(csrStart)
	fmt.Println("CSR+KEY TLS Duration: ", csrDuration)
	crtStart := time.Now()
	for _, ele := range nodeList {
		wg.Add(1)
		go genCert(ele, iteration, &wg)
	}
	wg.Wait()
	crtDuration := time.Since(crtStart)
	fmt.Println("CRT TLS Duration: ", crtDuration)
	duration := time.Since(start)
	fmt.Println("Total TLS Duration: ", duration)
}

func GenerateACLArtifacts(serviceMap []ACLMap) {
	fmt.Printf("Generating ACL artifacts for: %v\n", serviceMap)
}


func CSR(nodeName string, iteration int, wg *sync.WaitGroup) {
	semaphore <- struct{}{}
	keyBytes, _ := rsa.GenerateKey(rand.Reader, 4096)
	keyFile := "artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".key"
	pemfile, _ := os.Create(keyFile)
	var pemkey = &pem.Block{
                  Type : "RSA PRIVATE KEY",
                  Bytes : x509.MarshalPKCS1PrivateKey(keyBytes)}
	pem.Encode(pemfile, pemkey)
	pemfile.Close()

	subj := pkix.Name{
		CommonName:         nodeName,
		Country:            []string{"AU"},
		Province:           []string{"Some-State"},
		Locality:           []string{"MyCity"},
		Organization:       []string{"Company Ltd"},
		OrganizationalUnit: []string{"IT"},
	}

	template := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

    csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)
    fileName := "artifacts/"+strconv.Itoa(iteration)+"/"+nodeName+"/"+nodeName+".csr"
    clientCSRFile, err := os.Create(fileName)
    if err != nil {
        panic(err)
    }
    pem.Encode(clientCSRFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
    clientCSRFile.Close()
	<-semaphore
    defer wg.Done()
}



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
	extSubjectAltName := pkix.Extension{}
	extSubjectAltName.Id = asn1.ObjectIdentifier{2, 5, 29, 17}
	extSubjectAltName.Critical = false
	extSubjectAltName.Value = []byte("DNS:"+nodeName+"anvil.local")
	clientCRTTemplate.ExtraExtensions = []pkix.Extension{extSubjectAltName}

    certBytes, _ := x509.CreateCertificate(rand.Reader, clientCRTTemplate, caCRT, clientCSR.PublicKey, caPrivateKey)
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

