package main

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"os"
	"os/exec"
	"io"
	"path/filepath"
	"strings"
	//"bytes"
	"bufio"

	"strconv"
	"github.com/gorilla/mux"
)

type ACLMap struct {
        TokName         string
        Node            string
        Svc             string
        Valid	       []string
}

type FPMess struct {
	FilePath	string
}

var testMap []ACLMap

func main() {
    r := mux.NewRouter()
    registerRoutes(r)
    log.Fatal(http.ListenAndServe(":8080", r))
}

func registerRoutes(rot_router *mux.Router) {
    rot_router.HandleFunc("/bundle/{client_name}", RetrieveBundle).Methods("GET")
    rot_router.HandleFunc("/assignment", AssignedPortion).Methods("POST")
    rot_router.HandleFunc("/missing/{iter}", CollectAll).Methods("POST")
    rot_router.HandleFunc("/missingDirs/{iter}", CollectDirs).Methods("GET")
    rot_router.HandleFunc("/collectSignal", CollectSignal).Methods("POST")
    rot_router.HandleFunc("/makeCA", MakeCA).Methods("POST")
    rot_router.HandleFunc("/pullCA", PullCA).Methods("POST")
    rot_router.HandleFunc("/sendCA/{iter}/{name}", SendCA).Methods("GET")
    rot_router.HandleFunc("/", Index).Methods("GET")
}

func Index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Rotation Endpoint\n")
}

func RetrieveBundle(w http.ResponseWriter, req *http.Request) {
	bundle_target := mux.Vars(req)["client_name"]
	fmt.Fprint(w, "Bundle for " + bundle_target + " selected.\n")
	// Find the highest iteration num so far
	// Open this directory
	// Open the Client name directory
	// Serve the .tar.gz file within 
}

func CollectSignal(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	pullMap := struct {
		Targets		[]string
		Iteration	string
	}{}
	err = json.Unmarshal(b, &pullMap)
	if err != nil {
		log.Fatal()
	}

	// First get a list of all of my files and don't pull any from this node that I already made
	baseList := []string{}
	searchInd := "artifacts/"+pullMap.Iteration+"/"
        err = filepath.Walk("./"+searchInd,
            func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if !info.IsDir() {
                    fpath := strings.Split(path, searchInd)
                    baseList = append(baseList, fpath[1])
                }
            return nil
        })


	for _, t := range pullMap.Targets {
		go func(t string) {
			client := new(http.Client)
			//pReq, err := http.NewRequest("GET", "http://localhost:8080/missingDirs/"+pullMap.Iteration, nil)
			pReq, err := http.NewRequest("GET", "http://"+t+"/outbound/rotation/service/rotation/missingDirs/"+pullMap.Iteration, nil)
			resp, err := client.Do(pReq)

			b, err = ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			missMap := struct {
				Directories	[]string
				FPaths		[]string
			}{}
			err = json.Unmarshal(b, &missMap)
			if err != nil {
				log.Fatal()
			}

			// Make the directories that are in missMap
			for _, d := range missMap.Directories {
				newpath := filepath.Join(".", "artifacts", pullMap.Iteration, d)
				os.MkdirAll(newpath, os.ModePerm)
			}

			for _, f := range missMap.FPaths {
				fmt.Printf("Trying to pull %v from %v\n", f, t)
				/*
				fMess := &FPMess{FilePath: f}
				jsonData, err := json.Marshal(fMess)
				if err != nil {
					log.Fatalln("Unable to marshal JSON")
				}
				postVal := bytes.NewBuffer(jsonData)
				pReq, err = http.NewRequest("GET", "http://"+t+"/outbound/rotation/service/rotation/missing/"+pullMap.Iteration, postVal)
				resp, err := client.Do(pReq)
				*/

				if f == "acls.yaml" {
					fmt.Println("Doing special things because acls.yaml")
					/*
					defer resp.Body.Close()
					CombineACLs(pullMap.Iteration, resp.Body)
					*/
				} else {
					fmt.Println("Normal save stuff because: ", f)
					/*
					out, err := os.OpenFile("/root/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+f, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
					if err != nil  {
						fmt.Printf("FAILURE OPENING FILE\n")
					}
					defer out.Close()
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						fmt.Errorf("bad status: %s", resp.Status)
					}
					_, err = io.Copy(out, resp.Body)
					if err != nil  {
						fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
					}
					*/
				}
			}
		}(t)
	}
	fmt.Fprintf(w, "DONE\n")
}

/*
func prevMade(fp string, list []string) bool {
	for _,f := range list {
		if fp == f {
			fmt.Printf("I already have %v\n", fp)
			return true
		}
	}
	return false
}
*/

func CombineACLs(iter string, respCont io.ReadCloser) {
	fmt.Println("Found acls.yaml, trying to append to my existing file")
	f, err := os.OpenFile("artifacts/"+iter+"/acls.yaml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
	    panic(err)
	}

	scanner := bufio.NewScanner(respCont)
	scanner.Split(bufio.ScanLines)
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}
	fmt.Println("Looping through all lines and appending")
	for ind, each_ln := range text {
		if ind != 0 {
			if _, err = f.WriteString(each_ln + "\n"); err != nil {
			    panic(err)
			}
		}
	}
	defer f.Close()
}






func CollectAll(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Landed in CollectAll")
	iter := mux.Vars(req)["iter"]
	fmt.Println(iter)
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
	var filepath FPMess
        err = json.Unmarshal(b, &filepath)
        if err != nil {
                log.Fatal(err)
        }
	fmt.Println(filepath.FilePath)
	path := iter+"/"+filepath.FilePath
	w.Header().Set("Content-Type", "application/text")
	http.ServeFile(w, req, path)
}

func CollectDirs(w http.ResponseWriter, req *http.Request) {
	iter := mux.Vars(req)["iter"]
	dirMap := struct {
		Directories	[]string
		FPaths		[]string
	}{}

	topLvl, err := ioutil.ReadDir("./artifacts/"+iter)
	if err != nil {
		log.Println(err)
	}

	for _, f := range topLvl {
		if f.IsDir() {
			dirMap.Directories = append(dirMap.Directories, f.Name())
		}
	}

	searchInd := "artifacts/"+iter+"/"
	err = filepath.Walk("./"+searchInd,
	    func(path string, info os.FileInfo, err error) error {
	    if err != nil {
		return err
	    }
	    if !info.IsDir() {
		    fpath := strings.Split(path, searchInd)
		    dirMap.FPaths = append(dirMap.FPaths, fpath[1])
		}
	    return nil
	})
	if err != nil {
	    log.Println(err)
	}

	jsonData, err := json.Marshal(dirMap)
        if err != nil {
                log.Fatalln("Unable to marshal JSON")
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, string(jsonData))
}

func MakeCA(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        caContent := struct {
                Iteration	string
                QuorumMems	string
        }{}
        err = json.Unmarshal(b, &caContent)
        if err != nil {
                log.Fatal(err)
        }
	iter, _ := strconv.Atoi(caContent.Iteration)
	numQ, _ := strconv.Atoi(caContent.QuorumMems)
	CreateCAInfra(iter, numQ)
	fmt.Fprint(w, "OK\n")
}

func SendCA(w http.ResponseWriter, req *http.Request) {
	ca_target := mux.Vars(req)["name"]
	ca_iter := mux.Vars(req)["iter"]
	filepath := "/root/anvil-rotation/config/"+ca_iter+"/"+ca_target
	w.Header().Set("Content-Type", "application/text")
	http.ServeFile(w, req, filepath)
}

func PullCA(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        caContent := struct {
                Iteration	string
                Prefix		string
        }{}
        err = json.Unmarshal(b, &caContent)
        if err != nil {
                log.Fatal(err)
	}

	leaderIP := req.Header.Get("X-Forwarded-For")
	client := new(http.Client)
	for i:=0; i < 3; i++ {
		if i == 0 {
			out, err := os.OpenFile("/root/anvil/config/ca.crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()
			pReq, err := http.NewRequest("GET", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration+"/ca.crt", nil)
			resp, err := client.Do(pReq)
			if err != nil {
				fmt.Printf("FAILURE RETRIEVING FILE\n")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("bad status: %s", resp.Status)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil  {
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
		} else if i == 1 {
			out, err := os.OpenFile("/root/anvil/config/"+caContent.Prefix+".crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()
			pReq, err := http.NewRequest("GET", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration+"/"+caContent.Prefix+".crt", nil)
			resp, err := client.Do(pReq)
			if err != nil {
				fmt.Printf("FAILURE RETRIEVING FILE\n")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("bad status: %s", resp.Status)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil  {
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
			resp, err = client.Do(pReq)
		} else if i == 2 {
			out, err := os.OpenFile("/root/anvil/config/"+caContent.Prefix+".key", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()
			pReq, err := http.NewRequest("GET", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration+"/"+caContent.Prefix+".key", nil)
			resp, err := client.Do(pReq)
			if err != nil {
				fmt.Printf("FAILURE RETRIEVING FILE\n")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("bad status: %s", resp.Status)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil  {
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
			resp, err = client.Do(pReq)
		}
	}
        newpath := filepath.Join(".", "config", caContent.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
	_, err = exec.Command("/usr/bin/cp", "/root/anvil/config/ca.crt", "/root/anvil-rotation/config/"+caContent.Iteration+"/ca.crt").Output()
	if err != nil {
		fmt.Println(err)
	}
	_, err = exec.Command("/usr/bin/cp", "/root/anvil/config/"+caContent.Prefix+".crt", "/root/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".crt").Output()
	if err != nil {
		fmt.Println(err)
	}
	_, err = exec.Command("/usr/bin/cp", "/root/anvil/config/"+caContent.Prefix+".key", "/root/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".key").Output()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Fprint(w, "Notified Quorum\n")
}


func AssignedPortion(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	assignmentList := struct {
		Nodes		[]string
		SvcMap		[]ACLMap
		Gossip		bool
		Iteration	int
		Prefix		string
	}{}
	err = json.Unmarshal(b, &assignmentList)
	if err != nil {
		log.Fatal()
	}

	CreateDirectories(assignmentList.Iteration)
	if (assignmentList.Gossip == true) {
		GenerateUDPKey(assignmentList.Iteration)
	}
	GenerateTLSArtifacts(assignmentList.Nodes, assignmentList.Iteration, assignmentList.Prefix)
	GenerateACLArtifacts(assignmentList.SvcMap, assignmentList.Iteration)
	fmt.Fprint(w, "200 OK \r\n")
}
