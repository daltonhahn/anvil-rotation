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
	"io"
        "math/big"
        "time"
        "sync"
	"log"
        "math/rand"
        b64 "encoding/base64"
	"path/filepath"
	"archive/tar"
	"compress/gzip"
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
		//Compress files here
		fileCompress := []string{
			"config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".crt",
			"config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".key",
			"config/"+strconv.Itoa(iteration)+"/ca.crt",
		}
		// Create output file
		out, err := os.Create("config/"+strconv.Itoa(iteration)+"/server"+strconv.Itoa(i)+".tar.gz")
		if err != nil {
			log.Fatalln("Error writing archive:", err)
		}
		defer out.Close()

		// Create the archive and write the output to the "out" Writer
		err = createArchive(fileCompress, out)
		if err != nil {
			log.Fatalln("Error creating archive:", err)
		}

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


func createArchive(files []string, buf io.Writer) error {
	// Create new Writers for gzip and tar
	// These writers are chained. Writing to the tar writer will
	// write to the gzip writer which in turn will write to
	// the "buf" writer
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		err := addToArchive(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToArchive(tw *tar.Writer, filename string) error {
	// Open the file which will be written into the archive
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filename

	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to tar archive
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil
}
