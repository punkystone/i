package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Env struct {
	uploadsDirectory string
	maxFileAge       int
}

var (
	// the address to listen on
	address = "127.0.0.1:9005"
	// the directory to save the images in
	root = "/var/www/i.fourtf.com/"

	// maximum age for the files
	// the program will delete the files older than maxAge every 2 hours
	maxAge = time.Hour * 24 * 365
	// files to be ignored when deleting old files
	deleteIgnoreRegexp = regexp.MustCompile(`index\\.html|favicon\\.ico`)

	// length of the random filename
	randomAdjectivesCount = 2
	adjectives            = make([]string, 0)
	filetypes             = make(map[string]string)
)

func main() {
	env, err := checkEnv()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", env)
	b, err := os.ReadFile("./filetypes.json")

	if err == nil {
		data := make(map[string][]string)

		if err = json.Unmarshal(b, &data); err != nil {
			fmt.Println(err)
		} else {
			for val, keys := range data {
				for _, key := range keys {
					filetypes["."+strings.TrimLeft(key, ".")] = val
				}
			}
		}
	}

	fmt.Println(filetypes)

	file, err := os.Open("./adjectives1.txt")

	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(file)

	for {
		line, _, err := r.ReadLine()

		if err != nil {
			break
		}

		adjectives = append(adjectives, string(line))
	}

	go func() {
		for {
			<-time.After(time.Hour * 2)
			collectGarbage(env.uploadsDirectory, env.maxFileAge)
		}
	}()

	server := &http.Server{
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		Addr:         address,
	}

	// open http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, env.uploadsDirectory)
	})
	server.ListenAndServe()
}

func handleUpload(w http.ResponseWriter, r *http.Request, uploadsDirectory string) {
	defer r.Body.Close()

	infile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error parsing uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer infile.Close()

	filename := header.Filename
	var ext string

	// get extension from file name
	index := strings.LastIndex(filename, ".")

	if index == -1 {
		ext = ""
	} else {
		ext = filename[index:]
	}

	lastWord := "File"

	fmt.Println(ext)

	if val, ok := filetypes[ext]; ok {
		lastWord = strings.Title(val)
	}

	var savePath string
	var random string

	// find a random filename that doesn't exist already
	for i := 0; i < 100; i++ {
		for j := 0; j < randomAdjectivesCount; j++ {
			random += strings.TrimSpace(strings.Title(adjectives[rand.Intn(len(adjectives))]))
		}

		random += lastWord

		// fuck with link
		savePath = filepath.Join(uploadsDirectory, random, ext)

		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			break
		}
	}

	link := "https://" + r.Host + "/" + random + ext

	// save the file
	outfile, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "error while saving file: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err = io.Copy(outfile, infile)
	if err != nil {
		http.Error(w, "error while saving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	outfile.Close()

	// return the link as the http body
	w.Write([]byte(link))

	// do this or it doesn't work
	io.Copy(io.Discard, r.Body)
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

func collectGarbage(uploadsDirectory string, maxAge int) {
	files, err := os.ReadDir(uploadsDirectory)

	if err != nil {
		return
	}

	for _, file := range files {
		fname := file.Name()

		if file.IsDir() || deleteIgnoreRegexp.MatchString(fname) {
			continue
		}
		info, err := file.Info()
		if err != nil {
			fmt.Println(err)
			continue
		}
		if time.Since(info.ModTime()) > (time.Hour * time.Duration(maxAge)) {
			err := os.Remove(filepath.Join(uploadsDirectory, fname))

			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Printf("Removed %s \n", fname)
		}
	}
}
