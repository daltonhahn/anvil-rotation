package main

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"

	"strconv"
	"github.com/gorilla/mux"
)

type ACLMap struct {
        TokName         string
        Node            string
        Svc             string
        ValidList       []string
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
    rot_router.HandleFunc("/missing", CollectAll).Methods("GET")
    rot_router.HandleFunc("/", Index).Methods("GET")
}

func Index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Rotation Endpoint\n")

	assignmentList := make([]string, 10)
	for i := 1; i < 10; i++ {
	      assignmentList[i] = "test"+strconv.Itoa(i)
	}
	GenerateTLSArtifacts(assignmentList, 1)
}

func RetrieveBundle(w http.ResponseWriter, req *http.Request) {
	bundle_target := mux.Vars(req)["client_name"]
	fmt.Fprint(w, "Bundle for " + bundle_target + " selected.\n")
}

func CollectAll(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Sending all artifacts of current iteration\n")
}

func AssignedPortion(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	assignmentList := struct {
		Nodes	[]string
		SvcMap	[]ACLMap
		Iteration	int
	}{}
	err = json.Unmarshal(b, &assignmentList)
	if err != nil {
		log.Fatal()
	}
	/*
	assignmentList := make([]string, 1000)
	for i := 1; i < 1000; i++ {
	      assignmentList[i] = strconv.Itoa(i)
	}
	*/

	//fmt.Printf("%v\n", assignmentList.Nodes)
	//fmt.Printf("%v\n", assignmentList.SvcMap)
	//fmt.Printf("%v\n", assignmentList.Iteration)

	CreateDirectories(assignmentList.Iteration)
	//GenerateUDPKey(assignmentList.Iteration)
	GenerateTLSArtifacts(assignmentList.Nodes, assignmentList.Iteration)
	//GenerateACLArtifacts(assignmentList.SvcMap, assignmentList.Iteration)

	fmt.Fprint(w, "Getting assignments\n")
}
