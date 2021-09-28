package main

import (
	"time"
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"os"
	"os/exec"
	"errors"
	//"io"
	"path/filepath"
	"strings"
	"bytes"

	"strconv"
	"github.com/gorilla/mux"
	"github.com/avast/retry-go/v3"
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

type CollectMap struct {
	Target		string
	FilePath	string
}

var cMap []CollectMap

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
    rot_router.HandleFunc("/prepBundle", PrepBundle).Methods("POST")
    rot_router.HandleFunc("/collectSignal", CollectSignal).Methods("POST")
    rot_router.HandleFunc("/makeCA", MakeCA).Methods("POST")
    rot_router.HandleFunc("/pullCA", PullCA).Methods("POST")
    rot_router.HandleFunc("/fillCA", FillCA).Methods("POST")
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
		fmt.Println("Failing here?")
                log.Fatal(err)
		fmt.Println("YUP, FAILING HERE")
        }
	path := "/home/anvil/Desktop/anvil-rotation/artifacts/"+iter+"/"+filepath.FilePath
	w.Header().Set("Content-Type", "application/text")
	http.ServeFile(w, req, path)
}

func PrepBundle(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	pullMap := struct {
		Targets		[]string
		Iteration	string
	}{}
	err = json.Unmarshal(b, &pullMap)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Fatal()
	}
	cMap = []CollectMap{}

	for _, t := range pullMap.Targets {
		client := new(http.Client)
		pReq, err := http.NewRequest("GET", "http://"+t+"/outbound/rotation/service/rotation/missingDirs/"+pullMap.Iteration, nil)

		var body []byte
		err = retry.Do(
			func() error {
				resp, err := client.Do(pReq)
				if err != nil || resp.StatusCode != http.StatusOK {
					if err == nil {
						return errors.New("BAD STATUS CODE FROM SERVER")
					} else {
						return err
					}
				} else {
					defer resp.Body.Close()
					body, err = ioutil.ReadAll(resp.Body)
					if err != nil || resp.Body == nil || body == nil {
						return err
					}
					return nil
				}
			},
			retry.Attempts(3),
		)

		/*
		resp, err := client.Do(pReq)
		b, err = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		*/
		missMap := struct {
			Directories	[]string
			FPaths		[]string
		}{}
		err = json.Unmarshal(body, &missMap)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal()
		}

		for _, d := range missMap.Directories {
			newpath := filepath.Join("/home/anvil/Desktop/anvil-rotation", "artifacts", pullMap.Iteration, d)
			os.MkdirAll(newpath, os.ModePerm)
		}
		for _, fp := range missMap.FPaths {
			tempMap := CollectMap{Target: t, FilePath: fp}
			cMap = append(cMap, tempMap)
		}
	}
}

func CollectSignal(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	pullMap := struct {
		Iteration	string
		QuorumMems	[]string
	}{}
	err = json.Unmarshal(b, &pullMap)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Fatal()
	}

	for _, entry  := range cMap {
		fMess := &FPMess{FilePath: entry.FilePath}
		jsonData, err := json.Marshal(fMess)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatalln("Unable to marshal JSON")
		}
		postVal := bytes.NewBuffer(jsonData)
		client := new(http.Client)
		pReq, err := http.NewRequest("POST", "http://"+entry.Target+"/outbound/rotation/service/rotation/missing/"+pullMap.Iteration, postVal)
		var body []byte
		err = retry.Do(
			func() error {
				resp, err := client.Do(pReq)
				if err != nil || resp.StatusCode != http.StatusOK {
					if err == nil {
						return errors.New("BAD STATUS CODE FROM SERVER")
					} else {
						return err
					}
				} else {
					defer resp.Body.Close()
					body, err = ioutil.ReadAll(resp.Body)
					if err != nil {
						return err
					}
					return nil
				}
			},
			retry.Attempts(3),
		)
		/*
		resp, err := client.Do(pReq)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatalln("Unable to pull from other member")
		}
		defer resp.Body.Close()
		*/

		if entry.FilePath == "acls.yaml" {
			CombineACLs(pullMap.Iteration, body)
		} else {
			out, err := os.Create("/home/anvil/Desktop/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+entry.FilePath)
			if err != nil  {
				http.Error(w, err.Error(), 500)
				fmt.Printf("FAILURE OPENING FILE\n")
			}
			err = ioutil.WriteFile(out.Name(), body, 0755)
			if err != nil  {
				http.Error(w, err.Error(), 500)
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
			defer out.Close()

		}
	}

        newpath := filepath.Join("/home/anvil/Desktop/anvil/", "config/gossip", pullMap.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
        newpath = filepath.Join("/home/anvil/Desktop/anvil/", "config/acls", pullMap.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
        newpath = filepath.Join("/home/anvil/Desktop/anvil/", "config/certs", pullMap.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
	hname, _ := os.Hostname()
	exec.Command("sudo", "/bin/cp", "-f", "/home/anvil/Desktop/anvil-rotation/config/"+pullMap.Iteration+"/"+hname+".crt", "/home/anvil/Desktop/anvil/config/certs/"+pullMap.Iteration+"/"+hname+".crt").Output()

	src := "/home/anvil/Desktop/anvil-rotation/config/"+pullMap.Iteration+"/"+hname+".key"
	dest := "/home/anvil/Desktop/anvil/config/certs/"+pullMap.Iteration+"/"+hname+".key"

	bytesRead, err := ioutil.ReadFile(src)

	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(dest, bytesRead, 0644)

	if err != nil {
		log.Fatal(err)
	}
	//exec.Command("sudo", "/bin/cp", "-f", "/home/anvil/Destkop/anvil-rotation/config/"+pullMap.Iteration+"/"+hname+".key", "/home/anvil/Desktop/anvil/config/certs/"+pullMap.Iteration+"/"+hname+".key").Output()
	exec.Command("sudo", "/bin/cp", "-f", "/home/anvil/Desktop/anvil-rotation/artifacts/"+pullMap.Iteration+"/gossip.key", "/home/anvil/Desktop/anvil/config/gossip/"+pullMap.Iteration+"/gossip.key").Output()
	exec.Command("sudo", "/bin/cp", "-f", "/home/anvil/Desktop/anvil-rotation/artifacts/"+pullMap.Iteration+"/"+hname+"/acl.yaml", "/home/anvil/Desktop/anvil/config/acls/"+pullMap.Iteration+"/acl.yaml").Output()

	for _, ele := range pullMap.QuorumMems {
		exec.Command("sudo", "/bin/cp", "-f", "/home/anvil/Desktop/anvil-rotation/config/"+pullMap.Iteration+"/"+ele+".crt",
			"/home/anvil/Desktop/anvil/config/certs/"+pullMap.Iteration+"/"+ele+".crt").Output()
	}
	fmt.Fprintf(w, "DONE\n")
}

func CombineACLs(iter string, respCont []byte) {
	f, err := ioutil.ReadFile("/home/anvil/Desktop/anvil-rotation/artifacts/"+iter+"/acls.yaml")
	if err != nil {
	    panic(err)
	}
	var aclList []ACLMap
        err = yaml.Unmarshal(f, &aclList)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
	var compList []ACLMap
	/*
        f2, err := ioutil.ReadAll(respCont)
        if err != nil {
            panic(err)
        }
	*/
        err = yaml.Unmarshal(respCont, &compList)
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
	err = ioutil.WriteFile("/home/anvil/Desktop/anvil-rotation/artifacts/"+iter+"/acls.yaml", yamlOut, 0)
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
	path := "/home/anvil/Desktop/anvil-rotation/artifacts/"+iter+"/"+filepath.FilePath
	w.Header().Set("Content-Type", "application/text")
	http.ServeFile(w, req, path)
}

func CollectDirs(w http.ResponseWriter, req *http.Request) {
	iter := mux.Vars(req)["iter"]
	dirMap := struct {
		Directories	[]string
		FPaths		[]string
	}{}

	topLvl, err := ioutil.ReadDir("/home/anvil/Desktop/anvil-rotation/artifacts/"+iter)
	if err != nil {
		log.Println(err)
	}

	for _, f := range topLvl {
		if f.IsDir() {
			dirMap.Directories = append(dirMap.Directories, f.Name())
		}
	}

	searchInd := "/home/anvil/Desktop/anvil-rotation/artifacts/"+iter+"/"
	err = filepath.Walk(searchInd,
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
        iter := mux.Vars(req)["iter"]
        b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        var filepath FPMess
        err = json.Unmarshal(b, &filepath)
        if err != nil {
                log.Fatal(err)
        }
        path := "/home/anvil/Desktop/anvil-rotation/config/"+iter+"/"+filepath.FilePath
        w.Header().Set("Content-Type", "application/text")
        http.ServeFile(w, req, path)
}

func FillCA(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        caContent := struct {
                Iteration	string
		QuorumMems	[]string
        }{}
        err = json.Unmarshal(b, &caContent)
        if err != nil {
                log.Fatal(err)
	}
	baseList := []string{}
	searchInd := "/home/anvil/Desktop/anvil-rotation/artifacts/"+caContent.Iteration+"/"
        err = filepath.Walk(searchInd,
            func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if info.IsDir() && !strings.Contains(info.Name(), "server") {
                    baseList = append(baseList, info.Name())
                }
            return nil
        })
	for _, dirName := range baseList {
		for _, ele := range caContent.QuorumMems {
			cmd := exec.Command("/bin/cp", "-f", "/home/anvil/Desktop/anvil-rotation/config/"+caContent.Iteration+"/"+ele+".crt",
				"/home/anvil/Desktop/anvil-rotation/artifacts/"+caContent.Iteration+"/"+dirName+"/"+ele+".crt")
			err := cmd.Start()
			if err != nil {
				log.Println(err)
			}
			cmd.Wait()
		}
	}
}

func PullCA(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
        defer req.Body.Close()
        caContent := struct {
                Iteration	string
                Prefix		string
		QuorumMems	[]string
        }{}
        err = json.Unmarshal(b, &caContent)
        if err != nil {
                log.Fatal(err)
	}
	fmt.Printf("-------- PREFIX: %s\n", caContent.Prefix)
	leaderIP := req.Header.Get("X-Forwarded-For")
	client := new(http.Client)
        newpath := filepath.Join("/home/anvil/Desktop/anvil-rotation/", "config", caContent.Iteration)
        os.MkdirAll(newpath, os.ModePerm)
	for i:=0; i < 2; i++ {
		if i == 0 {
			out, err := os.OpenFile("/home/anvil/Desktop/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}

			fMess := &FPMess{FilePath: caContent.Prefix+".crt"}
                        jsonData, err := json.Marshal(fMess)
                        if err != nil {
                                log.Fatalln("Unable to marshal JSON")
                        }
                        postVal := bytes.NewBuffer(jsonData)
			pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

			var body []byte
			err = retry.Do(
				func() error {
					resp, err := client.Do(pReq)
					if err != nil || resp.StatusCode != http.StatusOK {
						if err == nil {
							return errors.New("BAD STATUS CODE FROM SERVER")
						} else {
							return err
						}
					} else {
						defer resp.Body.Close()
						body, err = ioutil.ReadAll(resp.Body)
						if err != nil {
							return err
						}
						return nil
					}
				},
				retry.Attempts(3),
			)

			/*
			resp, err := client.Do(pReq)
			if err != nil {
				fmt.Printf("FAILURE RETRIEVING FILE\n")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("bad status: %s", resp.Status)
			}
			*/
			err = ioutil.WriteFile(out.Name(), body, 0755)
			//_, err = io.Copy(out, body)
			if err != nil  {
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
			defer out.Close()
		} else if i == 1 {
			out, err := os.OpenFile("/home/anvil/Desktop/anvil-rotation/config/"+caContent.Iteration+"/"+caContent.Prefix+".key", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil  {
				fmt.Printf("FAILURE OPENING FILE\n")
			}

			fMess := &FPMess{FilePath: caContent.Prefix+".key"}
                        jsonData, err := json.Marshal(fMess)
                        if err != nil {
                                log.Fatalln("Unable to marshal JSON")
                        }
                        postVal := bytes.NewBuffer(jsonData)

			pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

			var body []byte
			err = retry.Do(
				func() error {
					resp, err := client.Do(pReq)
					if err != nil || resp.StatusCode != http.StatusOK {
						if err == nil {
							return errors.New("BAD STATUS CODE FROM SERVER")
						} else {
							return err
						}
					} else {
						defer resp.Body.Close()
						body, err = ioutil.ReadAll(resp.Body)
						if err != nil {
							return err
						}
						return nil
					}
				},
				retry.Attempts(3),
			)

			/*
			resp, err := client.Do(pReq)
			if err != nil {
				fmt.Printf("FAILURE RETRIEVING FILE\n")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("bad status: %s", resp.Status)
			}
			*/
			err = ioutil.WriteFile(out.Name(), body, 0755)
			//_, err = io.Copy(out, body)
			if err != nil  {
				fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
			}
			defer out.Close()
		}
	}
	for _, ele := range caContent.QuorumMems {
		out, err := os.OpenFile("/home/anvil/Desktop/anvil-rotation/config/"+caContent.Iteration+"/"+ele+".crt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil  {
			fmt.Printf("FAILURE OPENING FILE\n")
		}

		fMess := &FPMess{FilePath: ele+".crt"}
		jsonData, err := json.Marshal(fMess)
		if err != nil {
			log.Fatalln("Unable to marshal JSON")
		}
		postVal := bytes.NewBuffer(jsonData)
		pReq, err := http.NewRequest("POST", "http://"+leaderIP+"/outbound/rotation/service/rotation/sendCA/"+caContent.Iteration, postVal)

		var body []byte
                err = retry.Do(
                        func() error {
                                resp, err := client.Do(pReq)
                                if err != nil || resp.StatusCode != http.StatusOK {
                                        if err == nil {
                                                return errors.New("BAD STATUS CODE FROM SERVER")
                                        } else {
                                                return err
                                        }
                                } else {
                                        defer resp.Body.Close()
                                        body, err = ioutil.ReadAll(resp.Body)
                                        if err != nil {
                                                return err
                                        }
                                        return nil
                                }
                        },
			retry.Attempts(3),
                )

		/*
		resp, err := client.Do(pReq)
		if err != nil {
			fmt.Printf("FAILURE RETRIEVING FILE\n")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Errorf("bad status: %s", resp.Status)
		}
		*/
		err = ioutil.WriteFile(out.Name(), body, 0755)
		//_, err = io.Copy(out, body)
		if err != nil  {
			fmt.Printf("FAILURE WRITING OUT FILE CONTENTS\n")
		}
		defer out.Close()
	}
	fmt.Fprint(w, "Notified Quorum\n")
}


func AssignedPortion(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	assignmentList := struct {
		Quorum		[]string
		Nodes		[]string
		SvcMap		[]ACLMap
		Gossip		bool
		Iteration	int
		Prefix		string
	}{}
	err = json.Unmarshal(b, &assignmentList)
	if err != nil {
		log.Fatal(err)
	}

	CreateDirectories(assignmentList.Iteration)
	if (assignmentList.Gossip == true) {
		GenerateUDPKey(assignmentList.Iteration)
	}
	GenerateTLSArtifacts(assignmentList.Nodes, assignmentList.Iteration, assignmentList.Prefix, assignmentList.Quorum)
	GenerateACLArtifacts(assignmentList.SvcMap, assignmentList.Iteration)
	time.Sleep(5*time.Second)
	fmt.Fprint(w, "200 OK \r\n")
}
