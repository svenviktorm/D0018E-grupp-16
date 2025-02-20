// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		http.ServeFile(w, r, "website/start.html")
	} else {
		requestPath = requestPath[1:]
		requestPath = "website/" + requestPath
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
	_, ids, err := SearchBooksByTitleV1(requestData.Text)
	//fmt.Println(resp)
	var res string
	if err != nil {
		res = fmt.Sprintf("Error: %v\n", err)
	} else {
		res = fmt.Sprintf("Hits when searching for %v: %v\n", requestData.Text, ids)
	}
	// Create response
	response := ResponseData{Response: res}

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

func addBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var book Book
	err := json.NewDecoder(r.Body).Decode(&book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//json.Unmarshal([]byte(r), &book)

	id, err := AddBook(book)
	if err != nil {
		http.Error(w, "Failed to add book: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Received book: %+v\n", book)

	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Book added successfully",
		"id":      id,
	})
}

func viewBooksBySellerHandler(w http.ResponseWriter, r *http.Request) {
	sellerId := 1

	books, err := viewSellerBooks(sellerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var formattedBooks []map[string]interface{}

	for _, book := range books {
		formattedBooks = append(formattedBooks, map[string]interface{}{
			"title":       book.Title,
			"sellerid":    book.SellerID,
			"description": book.Description.String,
			"price":       book.Price,
			"edition":     book.Edition.String,
			"cathegory":   book.Cathegory,
			"stockAmount": book.StockAmount,
			"status":      book.Status,
		})
	}

	fmt.Printf("Books: %+v\n", formattedBooks)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"books":  formattedBooks,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "AnkaAnka",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "bookstore",
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

	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", viewHandler)
	http.HandleFunc("/add_book", addBookHandler)
	http.HandleFunc("/viewSellerBook", viewBooksBySellerHandler)
	//http.HandleFunc("POST /", viewHandler)
	fmt.Println("a!")
	http.HandleFunc("/root/", rootHandler)
	fmt.Println("b!")
	http.HandleFunc("/send", sendHandler)
	fmt.Println("c!")

	log.Fatal(http.ListenAndServe(":80", nil))
	fmt.Println("Server uppe!")
}
