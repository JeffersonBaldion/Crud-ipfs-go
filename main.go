package main

import (
	"net/http"

	"github.com/JeffersonBaldion/ipfsCRUD/routes"
	"github.com/gorilla/mux"
)

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", routes.HomeHandler)
	r.HandleFunc("/upload", routes.PutObject).Methods("POST")
	r.HandleFunc("/getFile", routes.GetObject).Methods("GET")
	r.HandleFunc("/deleteFile", routes.DeleteObject).Methods("DELETE")
	http.ListenAndServe(":3000", r)
}
