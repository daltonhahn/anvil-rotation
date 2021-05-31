package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"strings"
	"log"
)

var semaphore = make(chan struct{}, 250)

func CreateDirectories(iteration int) {
	newpath := filepath.Join(".", "artifacts", strconv.Itoa(iteration))
	os.MkdirAll(newpath, os.ModePerm)
}

func GenerateUDPKey(iteration int) {
	udpKey := (StringWithCharset(32, charset) + "\n")
	fileName := "artifacts/"+strconv.Itoa(iteration)+"/gossip.key"
	gossKeyFile, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	_, err = gossKeyFile.Write([]byte(udpKey))
	defer gossKeyFile.Close()
}

func GenerateTLSArtifacts(nodeList []string, iteration int) {
	//start := time.Now()
	var wg sync.WaitGroup
	newpath := filepath.Join(".", "artifacts", strconv.Itoa(iteration))
	for _, ele := range nodeList {
		nodePath := filepath.Join(newpath, ele)
		os.MkdirAll(nodePath, os.ModePerm)
	}
	//csrStart := time.Now()
	for _, ele := range nodeList {
		wg.Add(1)
		go CSR(ele, iteration, &wg)
	}
	wg.Wait()
	//csrDuration := time.Since(csrStart)
	//fmt.Println("CSR+KEY TLS Duration: ", csrDuration)
	//crtStart := time.Now()
	for _, ele := range nodeList {
		wg.Add(1)
		go genCert(ele, iteration, &wg)
	}
	wg.Wait()
	//crtDuration := time.Since(crtStart)
	//fmt.Println("CRT TLS Duration: ", crtDuration)
	//duration := time.Since(start)
	//fmt.Println("Total TLS Duration: ", duration)
}

func GenerateACLArtifacts(serviceMap []ACLMap, iteration int) {
	fmt.Printf("Generating ACL artifacts for: %v\n", serviceMap)
	fileName := "artifacts/"+strconv.Itoa(iteration)+"/acls.yaml"
        ACLFile, err := os.Create(fileName)
        if err != nil {
                panic(err)
        }
	var fullACLs strings.Builder
	fullACLs.WriteString("---\n")
	for _, ele := range serviceMap {
		fullACLs.WriteString("-\n")
		fullACLs.WriteString("  name: " + ele.TokName + "\n")
		fullACLs.WriteString("  sname: " + ele.Svc + "\n")
		tokVal := StringWithCharset(64, charset)
		fullACLs.WriteString("  val: " + tokVal + "\n")
		fullACLs.WriteString("  services:\n")
		for _,sname := range ele.ValidList {
			fullACLs.WriteString("    - " + sname + "\n")
		}
		nodeACL := "artifacts/"+strconv.Itoa(iteration)+"/"+ ele.Node + "/acl.yaml"
		f, err := os.OpenFile(nodeACL, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
		    log.Fatal(err)
		}
		var tempACL strings.Builder
		tempACL.WriteString("  -\n")
		tempACL.WriteString("    sname: " + ele.Svc + "\n")
		tempACL.WriteString("    tval: " + tokVal + "\n")
		_, err = f.Write([]byte(tempACL.String()))
		defer f.Close()
	}
        _, err = ACLFile.Write([]byte(fullACLs.String()))
        defer ACLFile.Close()
}
