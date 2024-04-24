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
	Buffer  string `json:"Buffer"`
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
		http.Error(w, "Error to decode Json response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func PutObject(w http.ResponseWriter, r *http.Request) {

	
	// Getting file from form
	fileIpfs, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error to get the form file", http.StatusBadRequest)
		return
	}
	defer fileIpfs.Close()

	// Creating body to upload the file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", r.FormValue("name"))
	if err != nil {
		fmt.Println("Error en partBody", err)
		return
	}

	_, err = io.Copy(part, fileIpfs)
	if err != nil {
		fmt.Println("Error in the body copy", err)
		return
	}
	writer.Close()

	// Creating Request POST

	urlAdd := "http://ec2-13-58-89-39.us-east-2.compute.amazonaws.com:5001/api/v0/add"
	req, err := http.NewRequest("POST", urlAdd, body)
	if err != nil {
		fmt.Println("Error to get request Post response", err)
		w.Write([]byte("Error to get request Post response"))
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Upload to Ipfs node by Ipfs API
	clientHttp := http.Client{}
	resp, err := clientHttp.Do(req)
	if err != nil {
		fmt.Println("Error to get Http response", err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error in responseBody", err)
		return
	}

	fmt.Println(string(responseBody))

	var response ipfsRespond
	erro := json.Unmarshal(responseBody, &response)
	if erro != nil {
		fmt.Println("Error to decode Json response:", erro)
		w.Write([]byte("Error to decode Json response"))

		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error to get the form file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Upload file to S3
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIA2UC3EP5C7AEVBSRC", "YYsa9DWWWEFdPVlHxjJFiyvwDG15GFgDf8wqQ/H4", "")),
	)
	if err != nil {
		fmt.Println("Error to load AWS configuration:", err)
		http.Error(w, "Error to load AWS configuration", http.StatusInternalServerError)
		return
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "hello-ipfs-node"

	key := response.Hash + r.FormValue("mime")

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		fmt.Println("Error to upload the file to S3:", err)
		http.Error(w, "Error to upload the file to S3", http.StatusInternalServerError)
		return
	}

	// Message response
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
	defer r.Body.Close() // Cerrar el cuerpo de la solicitud al finalizar la función

	cid := r.FormValue("cid")

	urlGet := "http://ec2-13-58-89-39.us-east-2.compute.amazonaws.com:5001/api/v0/block/stat?arg=" + cid
	urlBuffer := "http://ec2-13-58-89-39.us-east-2.compute.amazonaws.com:5001/api/v0/cat?arg=" + cid

	clientHttp := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := clientHttp.Post(urlGet, "", nil)
	if err != nil {
		fmt.Println("Error in HTTP request:", err)
		http.Error(w, "Error in HTTP request", http.StatusNotFound)
		return
	}

	defer resp.Body.Close() // Cerrar la respuesta HTTP al finalizar la función

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Cid not found", http.StatusNotFound)
		fmt.Println("Cid not found")
		return
	}

	respBuff, err := clientHttp.Post(urlBuffer, "", nil)
	if err != nil {
		fmt.Println("Error in cat Http request", err)
		http.Error(w, "Error in cat Http request", http.StatusNotFound)
		return
	}

	defer respBuff.Body.Close()

	bodyBuff, err := io.ReadAll(respBuff.Body)
	if err != nil {
		fmt.Println("Error to read the body buffer response:", err)
		http.Error(w, "Error to read the body buffer response", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error to read the body response:", err)
		http.Error(w, "Error to read the body response", http.StatusInternalServerError)
		return
	}

	var response getRespond
	erro := json.Unmarshal(body, &response)
	if erro != nil {
		fmt.Println("Error to decode Json response:", erro)
		http.Error(w, "Error to decode Json response", http.StatusInternalServerError)
		return
	}

	message := getRespond{Buffer: string(bodyBuff), Key: response.Key, Size: response.Size}

	jsonRespond, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error in bodyJson", err)
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonRespond)
}

func DeleteObject(w http.ResponseWriter, r *http.Request) {
	
	defer r.Body.Close() // Cerrar el cuerpo de la solicitud al finalizar la función

	cid := r.FormValue("cid")

	url := "http://ec2-13-58-89-39.us-east-2.compute.amazonaws.com:5001/api/v0/pin/rm?arg=" + cid

	resp, err := http.Post(url, "", nil)
	if err != nil {
		fmt.Println("Error: CID is not pinned:", err)
		w.Write([]byte("The pin has not been removed, The CID is not found"))
		return
	}
	defer resp.Body.Close() // Cerrar la respuesta HTTP al finalizar la función

	if resp.StatusCode != http.StatusOK {
		fmt.Println("The pin has not been removed")
		w.Write([]byte("The pin has not been removed, The CID is not found"))
		return
	}

	gcURL := "http://ec2-13-58-89-39.us-east-2.compute.amazonaws.com:5001/api/v0/repo/gc"
	req, err := http.NewRequest("POST", gcURL, nil)
	if err != nil {
		fmt.Println("Error creating the request", err)
		w.Write([]byte("Error creating the request"))
		return
	}

	clientHttp := http.Client{}
	resp, err = clientHttp.Do(req)
	if err != nil {
		fmt.Println("Error request execute", err)
		return
	}
	defer resp.Body.Close() // Cerrar la respuesta HTTP al finalizar la función

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error executing repo/gc", resp.StatusCode)
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIA2UC3EP5C7AEVBSRC", "YYsa9DWWWEFdPVlHxjJFiyvwDG15GFgDf8wqQ/H4", "")),
	)
	if err != nil {
		fmt.Println("Error to load AWS config:", err)
		http.Error(w, "Error to load AWS config", http.StatusInternalServerError)
		return
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "hello-ipfs-node"

	respList, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	})
	if err != nil {
		fmt.Println("Error to list objects in the bucket:", err)
		w.Write([]byte("Error to list objects in the bucket"))
		return
	}

	for _, obj := range respList.Contents {
		if strings.Contains(*obj.Key, r.FormValue("cid")) {

			_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
				Bucket: &bucketName,
				Key:    obj.Key,
			})
			if err != nil {
				fmt.Println("Error to delete S3 file:", err)
				w.Write([]byte("Error to delete S3 file"))
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
