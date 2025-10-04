package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Env struct {
	uploadsDirectory string
	maxFileAge       int
}

const idLength = 10
const cleanUpInterval = 2 * time.Hour

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	env, err := checkEnv()
	if err != nil {
		logger.Error("missing environment variables", "error", err)
		os.Exit(1)
	}
	fileTypesFile, err := os.Open("./filetypes.json")
	fileTypes := []string{}
	if err != nil {
		logger.Error("failed to open filetypes.json", "error", err)
	} else {
		err = json.NewDecoder(fileTypesFile).Decode(&fileTypes)
		if err != nil {
			logger.Error("failed to decode filetypes.json", "error", err)
		}
	}
	go func() {
		for {
			collectGarbage(logger, env.uploadsDirectory, env.maxFileAge)
			<-time.After(cleanUpInterval)
		}
	}()
	server := &http.Server{
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		Addr:         ":80",
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, logger, env.uploadsDirectory)
	})
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("failed to start server", "error ", err)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request, logger *slog.Logger, uploadsDirectory string) {
	defer r.Body.Close()
	uploadedFile, header, err := r.FormFile("file")
	if err != nil {
		logger.Error("failed to parse uploaded file", "error", err)
		http.Error(w, "error parsing uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer uploadedFile.Close()
	filename := header.Filename
	extension := ""
	index := strings.LastIndex(filename, ".")
	if index != -1 {
		extension = filename[index:]
	}
	idBytes := make([]byte, idLength)
	_, _ = rand.Read(idBytes)
	id := base64.RawURLEncoding.EncodeToString(idBytes)[:idLength]
	savePath := filepath.Join(uploadsDirectory, id) + extension
	savedFile, err := os.Create(savePath)
	if err != nil {
		logger.Error("error while creating file", "error", err)
		http.Error(w, "error while creating file: "+err.Error(), http.StatusBadRequest)
		return
	}
	_, err = io.Copy(savedFile, uploadedFile)
	if err != nil {
		logger.Error("error while saving file", "error", err)
		http.Error(w, "error while saving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer savedFile.Close()
	link := "https://" + r.Host + "/" + id + extension
	_, err = w.Write([]byte(link))
	if err != nil {
		logger.Error("error while writing response", "error", err)
	}
	_, err = io.Copy(io.Discard, r.Body)
	if err != nil {
		logger.Error("error while discarding request body", "error", err)
	}
}

func checkEnv() (*Env, error) {
	uploadsDirectory, exists := os.LookupEnv("UPLOADS_DIRECTORY")
	if !exists {
		return nil, errors.New("UPLOADS_DIRECTORY environment variable not set")
	}
	maxFileAge, exists := os.LookupEnv("MAX_FILE_AGE")
	if !exists {
		return nil, errors.New("MAX_FILE_AGE environment variable not set")
	}
	maxAgeInt, err := strconv.Atoi(maxFileAge)
	if err != nil {
		return nil, errors.New("MAX_FILE_AGE is not a valid integer")
	}
	env := &Env{
		uploadsDirectory: uploadsDirectory,
		maxFileAge:       maxAgeInt,
	}
	return env, nil
}

func collectGarbage(logger *slog.Logger, uploadsDirectory string, maxAge int) {
	files, err := os.ReadDir(uploadsDirectory)
	if err != nil {
		logger.Error("error reading directory", "error", err)
		return
	}
	for _, file := range files {
		filename := file.Name()
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			logger.Error("error getting file info", "error", err)
			continue
		}
		if time.Since(info.ModTime()) > (time.Hour * time.Duration(maxAge)) {
			err := os.Remove(filepath.Join(uploadsDirectory, filename))
			if err != nil {
				logger.Error("error removing file", "error", err)
				continue
			}
			logger.Info("removed old file", "info", filename)
		}
	}
}
