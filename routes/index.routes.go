package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	// "path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Message struct {
	Text string `json:"Welcome"`
}

type ipfsRespond struct {
	Hash string `json:"Hash"`
	Name string `json:"Name"`
	Size string `json:"Size"`
}

type putRespond struct {
	S3Check   bool        `json:"S3"`
	IPFSCheck bool        `json:"IPFS"`
	IpfsData  ipfsRespond `json:"IpfsData"`
}

type getRespond struct {
	Url  string `json:"Url"`
	Key  string `json:"Key"`
	Size int    `json:"Size"`
}

type deleteRespond struct {
	IPFS    bool   `json:"IPFS"`
	S3      bool   `json:"S3"`
	Message string `json:"Message"`
}
type Endpoint struct {
	Method    string            `json:"Methods"`
	Arguments map[string]string `json:"Arguments"`
	Respond   interface{}       `json:"respond"`
}

type ResponseData struct {
	IPFS    bool   `json:"IPFS"`
	S3      bool   `json:"S3"`
	Message string `json:"Message"`
}

type HomeResponse struct {
	Welcome   string              `json:"Welcome"`
	Important string              `json:"IMPORTANT"`
	Endpoints map[string]Endpoint `json:"Endpoints"`
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	response := HomeResponse{
		Welcome:   "This is a CRUD to handle a IPFS node and aws S3 storage",
		Important: "All arguments must be submitted in body request by a Form-data ",
		Endpoints: map[string]Endpoint{

			"/getFile": {
				Method: "GET",
				Arguments: map[string]string{
					"cid": "<CID>",
				},
				Respond: map[string]string{
					"Url":  "https://cloudflare-ipfs.com/ipfs/<CID>",
					"Key":  "<CID>",
					"Size": "<Size>",
				},
			},
			"/upload": {
				Method: "POST",
				Arguments: map[string]string{
					"file": "<File_Path>",
					"name": "<Any name>",
					"mime": "<File_Type>",
				},
				Respond: map[string]interface{}{
					"IPFS": true,
					"S3":   true,
					"IpfsData": map[string]string{
						"Hash": "<CID>",
						"Name": "<name>",
						"Size": "Size",
					},
				},
			},
			"/deleteFile": {
				Method: "DELETE",
				Arguments: map[string]string{
					"cid": "<CID>",
				},
				Respond: ResponseData{
					IPFS:    true,
					S3:      true,
					Message: "The file has been removed from IPFS node and S3 storage",
				},
			},
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error al codificar la respuesta JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func PutObject(w http.ResponseWriter, r *http.Request) {

	fileIpfs, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al obtener el archivo del formulario", http.StatusBadRequest)
		return
	}
	defer fileIpfs.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", r.FormValue("name"))
	if err != nil {
		fmt.Println("Error en partBody", err)
		return
	}

	_, err = io.Copy(part, fileIpfs)
	if err != nil {
		fmt.Println("Error en copy", err)
		return
	}
	writer.Close()

	urlAdd := "http://ec2-3-147-65-246.us-east-2.compute.amazonaws.com:5001/api/v0/add"
	req, err := http.NewRequest("POST", urlAdd, body)
	if err != nil {
		fmt.Println("Error en reqPost", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	clientHttp := http.Client{}
	resp, err := clientHttp.Do(req)
	if err != nil {
		fmt.Println("Error en respHttp", err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error en responseBody", err)
		return
	}

	fmt.Println(string(responseBody))

	var response ipfsRespond
	erro := json.Unmarshal(responseBody, &response)
	if erro != nil {
		fmt.Println("Error al decodificar la respuesta JSON:", erro)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al obtener el archivo del formulario", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 	Subir archivo a S3

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIAYS2NVXW5WLWN2TX5", "sEuxwRGHkT0feyHYg2VdyZ8WCYPF21qdKfrwNTIk", "")),
	)
	if err != nil {
		fmt.Println("Error al cargar la configuración de AWS:", err)
		http.Error(w, "Error al cargar la configuración de AWS", http.StatusInternalServerError)
		return
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "jeff-test-ipfs-bucket"

	key := response.Hash + r.FormValue("mime")

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{

		Bucket: &bucketName,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		fmt.Println("Error al subir archivo a S3:", err)
		http.Error(w, "Error al subir archivo a S3", http.StatusInternalServerError)
		return
	}

	// Mensaje de respuesta

	message := putRespond{S3Check: true, IPFSCheck: true, IpfsData: ipfsRespond{
		Hash: response.Hash,
		Name: response.Name,
		Size: response.Size,
	}}

	jsonData, err := json.Marshal(message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func GetObject(w http.ResponseWriter, r *http.Request) {

	cid := r.FormValue("cid")

	urlGet := "http://ec2-3-147-65-246.us-east-2.compute.amazonaws.com:5001/api/v0/block/stat?arg=" + cid

	clientHttp := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := clientHttp.Post(urlGet, "", nil)
	if err != nil {
		fmt.Println("Error al hacer la solicitud HTTP:", err)
		http.Error(w, "Cid not found", http.StatusNotFound)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Cid not found", http.StatusNotFound)
		fmt.Println("Cid not found")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error al leer la respuesta:", err)
		return
	}

	var response getRespond
	erro := json.Unmarshal(body, &response)
	if erro != nil {
		fmt.Println("Error al decodificar la respuesta JSON:", erro)
		return
	}
	urlToSearch := "https://cloudflare-ipfs.com/ipfs/" + response.Key

	message := getRespond{Url: urlToSearch, Key: response.Key, Size: response.Size}

	jsonRespond, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error en bodyJson", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonRespond)
}

func DeleteObject(w http.ResponseWriter, r *http.Request) {

	cid := r.FormValue("cid")

	url := "http://ec2-3-147-65-246.us-east-2.compute.amazonaws.com:5001/api/v0/pin/rm?arg=" + cid

	resp, err := http.Post(url, "", nil)
	if err != nil {
		fmt.Println("Error: CID is not pinned:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("The pin has not been removed")
		return
	}

	gcURL := "http://ec2-3-147-65-246.us-east-2.compute.amazonaws.com:5001/api/v0/repo/gc"
	req, err := http.NewRequest("POST", gcURL, nil)
	if err != nil {
		fmt.Println("Error creating the request", err)
		return
	}

	clientHttp := http.Client{}
	resp, err = clientHttp.Do(req)
	if err != nil {
		fmt.Println("Error request execute", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error executing repo/gc", resp.StatusCode)
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIAYS2NVXW5WLWN2TX5", "sEuxwRGHkT0feyHYg2VdyZ8WCYPF21qdKfrwNTIk", "")),
	)
	if err != nil {
		fmt.Println("Error al cargar la configuración de AWS:", err)
		http.Error(w, "Error al cargar la configuración de AWS", http.StatusInternalServerError)
		return
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "jeff-test-ipfs-bucket"

	respList, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	})
	if err != nil {
		fmt.Println("error al listar objetos en el bucket:", err)
		return
	}

	for _, obj := range respList.Contents {
		if strings.Contains(*obj.Key, r.FormValue("cid")) {
			// Eliminar el objeto
			_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
				Bucket: &bucketName,
				Key:    obj.Key,
			})
			if err != nil {
				fmt.Println("error al eliminar el archivo de S3:", err)
				return
			}
		}
	}

	message := deleteRespond{IPFS: true, S3: true, Message: "The file has been removed from IPFS node and S3 storage"}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error creating json response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonMessage)

}
