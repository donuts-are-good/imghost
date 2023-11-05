package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"github.com/google/uuid"
)

type Config struct {
	SecretKey          string   `json:"secretKey"`
	ImageDirectory     string   `json:"imageDirectory"`
	ImageUrl           string   `json:"imageUrl"`
	Port               string   `json:"port"`
	ResizeWidth        int      `json:"resizeWidth"`
	ResizeHeight       int      `json:"resizeHeight"`
	CropWidth          int      `json:"cropWidth"`
	CropHeight         int      `json:"cropHeight"`
	ImageFormat        string   `json:"imageFormat"`
	UploadRoute        string   `json:"uploadRoute"`
	AllowedIPs         []string `json:"allowedIPs"`
	LogFilePath        string   `json:"logFilePath"`
	GenerateThumbnails bool     `json:"generateThumbnails"`
	CheckIP            bool     `json:"checkIP"`
}

var config = Config{
	SecretKey:          "your-secret-key",
	ImageDirectory:     "path/to/your/images/directory",
	ImageUrl:           "http://your-domain.com/path/to/your/images/directory",
	Port:               "39716",
	ResizeWidth:        512,
	ResizeHeight:       512,
	CropWidth:          512,
	CropHeight:         512,
	ImageFormat:        "png",
	UploadRoute:        "/upload",
	AllowedIPs:         []string{"159.203.109.32", "203.0.1.0"},
	LogFilePath:        "imghost.log",
	GenerateThumbnails: true,
	CheckIP:            true,
}
var logger *log.Logger

func main() {
	// check if the config file exists
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		// if it doesn't exist, create it with demo values
		file, err := os.Create("config.json")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "\t")
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

	log.Println("imghost running on port: ", config.Port)

	// check if the log file exists
	if _, err := os.Stat(config.LogFilePath); err == nil {
		// if it exists, rename it
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

	// create a new serve mux
	mux := http.NewServeMux()

	// serve static files from the ImageDirectory
	fs := http.FileServer(http.Dir(config.ImageDirectory))
	mux.Handle("/img/", logHandler(http.StripPrefix("/img/", fs)))

	// handle both /upload and /upload/ routes
	mux.Handle(config.UploadRoute, corsHandler(http.HandlerFunc(uploadHandler)))
	if !strings.HasSuffix(config.UploadRoute, "/") {
		mux.Handle(config.UploadRoute+"/", corsHandler(http.HandlerFunc(uploadHandler)))
	}

	http.ListenAndServe(":"+config.Port, mux)
}

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Request received: %s %s", r.Method, r.URL.Path)
		start := time.Now()
		h.ServeHTTP(w, r)
		logger.Printf("Request processed in %s", time.Since(start))
	})
}

func corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	if config.CheckIP {
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
			logger.Println("Rejected IP: ", ip)
			http.Error(w, "Not allowed", http.StatusForbidden)
			return
		}
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
	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Could not read image file", http.StatusInternalServerError)
		return
	}

	// log the original image details
	logger.Printf("Datetime: %s, Original filename: %s, Original dimensions: %dx%d",
		time.Now().Format(time.RFC3339), header.Filename, img.Bounds().Dx(), img.Bounds().Dy())

	// calculate aspect ratio
	aspectRatio := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())

	// calculate new dimensions while maintaining aspect ratio
	newWidth := config.ResizeWidth
	newHeight := int(float64(newWidth) / aspectRatio)
	if newHeight < config.ResizeHeight {
		newHeight = config.ResizeHeight
		newWidth = int(float64(newHeight) * aspectRatio)
	}

	// resize image while maintaining aspect ratio
	resized := transform.Resize(img, newWidth, newHeight, transform.Linear)

	// calculate crop rectangle
	cropX := (resized.Bounds().Dx() - config.CropWidth) / 2
	cropY := (resized.Bounds().Dy() - config.CropHeight) / 2
	cropRect := image.Rect(cropX, cropY, cropX+config.CropWidth, cropY+config.CropHeight)

	// crop image
	cropped := transform.Crop(resized, cropRect)

	// generate a UUID for the filename
	filename := uuid.New().String()

	// write image to file
	err = imgio.Save(fmt.Sprintf("%s/%s.%s", config.ImageDirectory, filename, config.ImageFormat), cropped, imgio.PNGEncoder())
	if err != nil {
		http.Error(w, "Could not write image file", http.StatusInternalServerError)
		return
	}

	// if GenerateThumbnails is true, generate a thumbnail
	if config.GenerateThumbnails {
		thumbnail := transform.Resize(cropped, 64, 64, transform.Linear)
		err = imgio.Save(fmt.Sprintf("%s/%s_thumbnail.%s", config.ImageDirectory, filename, config.ImageFormat), thumbnail, imgio.PNGEncoder())
		if err != nil {
			http.Error(w, "Could not write thumbnail file", http.StatusInternalServerError)
			return
		}
	}

	// log the final image details
	logger.Printf("Datetime: %s, Final filename: %s.%s, Final dimensions: %dx%d",
		time.Now().Format(time.RFC3339), filename, config.ImageFormat, cropped.Bounds().Dx(), cropped.Bounds().Dy())

	// return the image URL
	w.Write([]byte(fmt.Sprintf("%s/%s.%s", config.ImageUrl, filename, config.ImageFormat)))
}
