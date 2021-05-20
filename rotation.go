package main

import (
	"fmt"
	"net/http"
	"log"
	"github.com/gorilla/mux"
)


func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", Index)
    log.Fatal(http.ListenAndServe(":8080", r))
}

func Index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello from rotation\n")
}
