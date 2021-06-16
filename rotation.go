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
	"bytes"
	"time"

	"strconv"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type ACLMap struct {
	TokName         string		`yaml:"name,omitempty"`
        Node            string		`yaml:"sname,omitempty"`
        Svc             string		`yaml:"val,omitempty"`
        Valid	       []string		`yaml:"services,omitempty"`
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
    rot_router.HandleFunc("/bundle/{iter}", RetrieveBundle).Methods("POST")
    rot_router.HandleFunc("/assignment", AssignedPortion).Methods("POST")
    rot_router.HandleFunc("/missing/{iter}", CollectAll).Methods("POST")
    rot_router.HandleFunc("/missingDirs/{iter}", CollectDirs).Methods("GET")
    rot_router.HandleFunc("/collectSignal", CollectSignal).Methods("POST")
    rot_router.HandleFunc("/makeCA", MakeCA).Methods("POST")
    rot_router.HandleFunc("/pullCA", PullCA).Methods("POST")
    rot_router.HandleFunc("/sendCA/{iter}", SendCA).Methods("POST")
    rot_router.HandleFunc("/", Index).Methods("GET")
}

func Index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Rotation Endpoint\n")
}

func RetrieveBundle(w http.ResponseWriter, req *http.Request) {
	iter := mux.Vars(req)["iter"]
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	var filepath FPMess
        err = json.Unmarshal(b, &filepath)
        if err != nil {
                log.Fatal(err)
        }
	path := "/root/anvil-rotation/artifacts/"+iter+"/"+filepath.FilePath
	w.Header().Set("Content-Type", "application/text")
	http.ServeFile(w, req, path)
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

			for _, d := range missMap.Directories {
				newpath := filepath.Join(".", "artifacts", pullMap.Iteration, d)
				os.MkdirAll(newpath, os.ModePerm)
			}

			for _, f := range missMap.FPaths {
				fMess := &FPMess{FilePath: f}
				jsonData, err := json.Marshal(fMess)
				if err != nil {
					log.Fatalln("Unable to marshal JSON")
				}
				postVal := bytes.NewBuffer(jsonData)
				pReq, err = http.NewRequest("POST", "http://"+t+"/outbound/rotation/service/rotation/missing/"+pullMap.Iteration, postVal)
				resp, err := client.Do(pReq)

				if f == "acls.yaml" {
					defer resp.Body.Close()
					CombineACLs(pullMap.Iteration, resp.Body)
				} else {
					out, err := os.Create("/root/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+f)
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

				}
			}
		}(t)
	}

        newpath := filepath.Join("/root/anvil/", "config/acls", pullMap.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
        newpath = filepath.Join("/root/anvil/", "config/certs", pullMap.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
	hname, _ := os.Hostname()
	_, err = exec.Command("/usr/bin/cp", "/root/anvil-rotation/config/"+pullMap.Iteration+"/ca.crt", "/root/anvil/config/certs/"+pullMap.Iteration+"/ca.crt").Output()
	if err != nil {
		fmt.Println(err)
	}
	_, err = exec.Command("/usr/bin/cp", "/root/anvil-rotation/config/"+pullMap.Iteration+"/"+hname+".crt", "/root/anvil/config/certs/"+pullMap.Iteration+"/"+hname+".crt").Output()
	if err != nil {
		fmt.Println(err)
	}
	_, err = exec.Command("/usr/bin/cp", "/root/anvil-rotation/config/"+pullMap.Iteration+"/"+hname+".key", "/root/anvil/config/certs/"+pullMap.Iteration+"/"+hname+".key").Output()
	if err != nil {
		fmt.Println(err)
	}
	exec.Command("/usr/bin/cp", "/root/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+hname+"/acl.yaml", "/root/anvil/config/acls/"+pullMap.Iteration+"/acl.yaml").Output()
	fmt.Fprintf(w, "DONE\n")
}

func CombineACLs(iter string, respCont io.ReadCloser) {
	f, err := ioutil.ReadFile("artifacts/"+iter+"/acls.yaml")
	if err != nil {
	    panic(err)
	}
	var aclList []ACLMap
        err = yaml.Unmarshal(f, &aclList)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
	var compList []ACLMap
        f2, err := ioutil.ReadAll(respCont)
        if err != nil {
            panic(err)
        }
        err = yaml.Unmarshal(f2, &compList)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }

	retList := make([]ACLMap, len(aclList))
	copy(retList, aclList)
	for _, ele := range compList {
		if !valInList(ele, aclList) {
			retList = append(retList, ele)
		}
	}
	yamlOut,err := yaml.Marshal(&retList)
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile("artifacts/"+iter+"/acls.yaml", yamlOut, 0)
	if err != nil {
		fmt.Println(err)
	}
}

func valInList(a ACLMap, list []ACLMap) bool {
    for _, ele := range list {
        if ele.TokName == a.TokName {
            return true
        }
    }
    return false
}


func CollectAll(w http.ResponseWriter, req *http.Request) {
	iter := mux.Vars(req)["iter"]
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
	var filepath FPMess
        err = json.Unmarshal(b, &filepath)
        if err != nil {
                log.Fatal(err)
        }
	path := "/root/anvil-rotation/artifacts/"+iter+"/"+filepath.FilePath
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
	time.Sleep(3*time.Second)
	fmt.Fprint(w, "OK\n")
}

func SendCA(w http.ResponseWriter, req *http.Request) {
        iter := mux.Vars(req)["iter"]
        b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        var filepath FPMess
        err = json.Unmarshal(b, &filepath)
        if err != nil {
                log.Fatal(err)
        }
        path := "/root/anvil-rotation/config/"+iter+"/"+filepath.FilePath
        w.Header().Set("Content-Type", "application/text")
        http.ServeFile(w, req, path)
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
        newpath := filepath.Join(".", "config", caContent.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
	for i:=0; i < 3; i++ {
		if i == 0 {
			out, err := os.OpenFile("/root/anvil-rotation/config/"+caContent.Iteration+"/ca.crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()

			fMess := &FPMess{FilePath: "ca.crt"}
			jsonData, err := json.Marshal(fMess)
			if err != nil {
				log.Fatalln("Unable to marshal JSON")
			}
			postVal := bytes.NewBuffer(jsonData)
			pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

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
			out, err := os.OpenFile("/root/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()

			fMess := &FPMess{FilePath: caContent.Prefix+".crt"}
                        jsonData, err := json.Marshal(fMess)
                        if err != nil {
                                log.Fatalln("Unable to marshal JSON")
                        }
                        postVal := bytes.NewBuffer(jsonData)
			pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

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
			out, err := os.OpenFile("/root/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".key", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			defer out.Close()

			fMess := &FPMess{FilePath: caContent.Prefix+".key"}
                        jsonData, err := json.Marshal(fMess)
                        if err != nil {
                                log.Fatalln("Unable to marshal JSON")
                        }
                        postVal := bytes.NewBuffer(jsonData)
			pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

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
