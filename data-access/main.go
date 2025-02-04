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

func loadPage(title string) (*Page, error) {
        filename := title + ".txt"
        body, err := os.ReadFile(filename)
        if err != nil {
                return nil, err
        }
        return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("view:")
    fmt.Println(w)
    fmt.Println("")
    fmt.Println(r)
    fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", w, r)
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
    //fmt.Println("root:")
    //fmt.Println(w)
    //fmt.Println("")
    //fmt.Println(r)
    title := r.URL.Path[len("/view/"):]
    fmt.Println(title)
    //p, _ := loadPage("html")
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
    resp, err := albumsByArtist(requestData.Text)
    fmt.Println(resp)
    res := fmt.Sprintf("Albums found: %v\n", resp)
	// Create response
	response:= ResponseData{Response: res}
    
	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
        DBName: "recordings",
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
    albums, err := albumsByArtist("SELECT * FROM album WHERE artist = 'John Coltrane'")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Albums found: %v\n", albums)
    http.HandleFunc("/view/", viewHandler)
    fmt.Println("a!")
    http.HandleFunc("/root/", rootHandler)
    fmt.Println("b!")
    http.HandleFunc("/send", sendHandler)
    fmt.Println("c!")
    log.Fatal(http.ListenAndServe(":80", nil))
    fmt.Println("Server uppe!")
}

// albumsByArtist queries for albums that have the specified artist name.
func albumsByArtist(name string) ([]Album, error) {
        // An albums slice to hold data from returned rows.
        var albums []Album
        rows, err := db.Query(name)
        if err != nil {
            return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
        }
        defer rows.Close()
        // Loop through rows, using Scan to assign column data to struct fields.
        for rows.Next() {
            var alb Album
            if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
                return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
            }
            albums = append(albums, alb)
        }
        if err := rows.Err(); err != nil {
            return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
        }
return albums, nil
}