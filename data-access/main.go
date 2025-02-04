// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
        "fmt"
        "encoding/json"
	    "database/sql"
        "log"
        "net/http"
        "os"

	"github.com/go-sql-driver/mysql"
)

type RequestData struct {
	Text string `json:"text"`
}

// Struct to send a JSON response
type ResponseData struct {
	Response string `json:"response"`
}

type Page struct {
        Title string
        Body  []byte
}

func (p *Page) save() error {
        filename := p.Title + ".txt"
        return os.WriteFile(filename, p.Body, 0600)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
    requestPath := r.URL.Path
    fmt.Println(requestPath)
    if requestPath == "/" {
        http.ServeFile(w, r, "start.html")
    } else {
        requestPath = requestPath[1:]
        requestPath = "website/"+requestPath
        http.ServeFile(w, r, requestPath)
    }
    
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html.html")
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process the input text (modify response as needed)
	responseText := fmt.Sprintf("You sent: %s", requestData.Text)
    fmt.Println(responseText)
	// Create response
    
	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseText)
}

var db *sql.DB

type Album struct {
        ID     int64
        Title  string
        Artist string
        Price  float32
    }

func main() {
    // Capture connection properties.
    cfg := mysql.Config{
        User:   "root",
        Passwd: "AnkaAnka",
        Net:    "tcp",
        Addr:   "127.0.0.1:3306",
        DBName: "bookstore",
        AllowNativePasswords: true,
    }
    // Get a database handle.
    var err error
    db, err = sql.Open("mysql", cfg.FormatDSN())
    if err != nil {
        log.Fatal(err)
    }

    pingErr := db.Ping()
    if pingErr != nil {
        log.Fatal(pingErr)
    }
    fmt.Println("Connected!")

    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    http.HandleFunc("/", viewHandler)
    fmt.Println("a!")
    http.HandleFunc("/root/", rootHandler)
    fmt.Println("b!")
    http.HandleFunc("/send", sendHandler)
    fmt.Println("c!")
    log.Fatal(http.ListenAndServe(":80", nil))
    fmt.Println("Server uppe!")
}