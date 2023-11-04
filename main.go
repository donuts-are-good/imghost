package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/gographics/imagick.v2/imagick"
)

type Config struct {
	SecretKey      string   `json:"secretKey"`
	ImageDirectory string   `json:"imageDirectory"`
	ImageUrl       string   `json:"imageUrl"`
	Port           string   `json:"port"`
	ResizeWidth    uint     `json:"resizeWidth"`
	ResizeHeight   uint     `json:"resizeHeight"`
	CropWidth      uint     `json:"cropWidth"`
	CropHeight     uint     `json:"cropHeight"`
	ImageFormat    string   `json:"imageFormat"`
	UploadRoute    string   `json:"uploadRoute"`
	AllowedIPs     []string `json:"allowedIPs"`
	LogFilePath    string   `json:"logFilePath"`
}

var config Config
var logger *log.Logger

func main() {
	// check if the config file exists
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		// if it doesn't exist, create it with demo values
		config = Config{
			SecretKey:      "your-secret-key",
			ImageDirectory: "path/to/your/images/directory",
			ImageUrl:       "http://your-domain.com/path/to/your/images/directory",
			Port:           "8080",
			ResizeWidth:    512,
			ResizeHeight:   512,
			CropWidth:      512,
			CropHeight:     512,
			ImageFormat:    "png",
			UploadRoute:    "/upload",
			AllowedIPs:     []string{"192.0.2.1", "203.0.113.42"},
			LogFilePath:    "imghost.log",
		}
		file, err := os.Create("config.json")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(config)

		if err != nil {
			panic(err)
		}
	} else {
		// if it exists, read and decode it
		file, err := os.ReadFile("config.json")
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(file, &config)
		if err != nil {
			panic(err)
		}
	}

	// check if the log file exists
	if _, err := os.Stat(config.LogFilePath); err == nil {
		// If it exists, rename it
		err = os.Rename(config.LogFilePath, fmt.Sprintf("%s.%s.old", config.LogFilePath, time.Now().Format("2006-01-02")))
		if err != nil {
			panic(err)
		}
	}

	// open the log file
	logFile, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// create a logger that writes to the log file and the standard output
	logger = log.New(io.MultiWriter(logFile, os.Stdout), "", log.LstdFlags)

	http.HandleFunc(config.UploadRoute, uploadHandler)
	http.ListenAndServe(":"+config.Port, nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// check the client's IP
	ip := strings.Split(r.RemoteAddr, ":")[0]
	isAllowed := false
	for _, allowedIP := range config.AllowedIPs {
		if ip == allowedIP {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		http.Error(w, "Not allowed", http.StatusForbidden)
		return
	}

	// check the secret key
	if r.FormValue("key") != config.SecretKey {
		http.Error(w, "Invalid secret key", http.StatusUnauthorized)
		return
	}

	// get the image file
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Could not get image file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// read the image file
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read image file", http.StatusInternalServerError)
		return
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// read image from data
	err = mw.ReadImageBlob(data)
	if err != nil {
		http.Error(w, "Could not read image blob", http.StatusInternalServerError)
		return
	}

	// log the original image details
	logger.Printf("Datetime: %s, Original filename: %s, Original dimensions: %dx%d, Original filesize: %d bytes",
		time.Now().Format(time.RFC3339), header.Filename, mw.GetImageWidth(), mw.GetImageHeight(), len(data))

	// resize image
	err = mw.ResizeImage(config.ResizeWidth, config.ResizeHeight, imagick.FILTER_LANCZOS, 1)
	if err != nil {
		http.Error(w, "Could not resize image", http.StatusInternalServerError)
		return
	}

	// crop image
	err = mw.CropImage(config.CropWidth, config.CropHeight, int(mw.GetImageWidth()-config.CropWidth)/2, int(mw.GetImageHeight()-config.CropHeight)/2)
	if err != nil {
		http.Error(w, "Could not crop image", http.StatusInternalServerError)
		return
	}

	// generate a UUID for the filename
	filename := uuid.New().String()

	// write image to file
	err = mw.WriteImage(fmt.Sprintf("%s/%s.%s", config.ImageDirectory, filename, config.ImageFormat))
	if err != nil {
		http.Error(w, "Could not write image file", http.StatusInternalServerError)
		return
	}

	// log the final image details
	logger.Printf("Datetime: %s, Final filename: %s.%s, Final dimensions: %dx%d, Final filesize: %d bytes",
		time.Now().Format(time.RFC3339), filename, config.ImageFormat, mw.GetImageWidth(), mw.GetImageHeight(), len(mw.GetImageBlob()))

	// return the image URL
	w.Write([]byte(fmt.Sprintf("%s/%s.%s", config.ImageUrl, filename, config.ImageFormat)))
}
