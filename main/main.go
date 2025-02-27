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
	"strconv"

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
	fmt.Println("viewHandler called")
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
	fmt.Println("rootHandler called")
	http.ServeFile(w, r, "html.html")
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("userHandler called")
	switch r.Method {
	case http.MethodGet:
		fmt.Println("Get request to users API")
		fmt.Println("This should be an attempt to login or similar")
		//fmt.Println(r)
		//r.ParseForm()
		//fmt.Println(r.Form)
		uname := r.FormValue("username")
		pwd := r.FormValue("password")
		fmt.Printf("username:%v, password:%v, hash:%v", uname, pwd, hash(pwd))
		fmt.Println("")
		//fmt.Println(r.Form)

		user, loginOK, err := LogInCheckNotHashed(uname, pwd)
		user.Password = pwd
		fmt.Printf("login ok?:%v, username: %v userID:%v seller?:%v, admin?:%v ", loginOK, user.Username, user.UserID, user.IsSeller, user.IsAdmin)

		fmt.Println("")
		if loginOK {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		} else {
			if err != nil {
				//http.Error(w, fmt.Printf("Some error occured: %v", err), http.StatusInternalServerError)
				http.Error(w, "Some error occured", http.StatusInternalServerError)
				//TODO fix this
			} else {
				http.Error(w, "Invalid username or password", http.StatusNotFound)
			}
		}
	case http.MethodPost:
		fmt.Println("Post request to users API")
		fmt.Println("This should be an attempt to create a user account")

		username := r.FormValue("username")
		pwd := r.FormValue("password")
		email := r.FormValue("email")
		fmt.Println("username:%v, password:%v, mail:%v", username, pwd, email)
		emailSQL := sql.NullString{String: email, Valid: true}
		if email == "" {
			emailSQL = sql.NullString{String: "", Valid: false}
		}
		id, err := AddUser(username, pwd, emailSQL)
		if err != nil {
			fmt.Println("Failed to add user: ", err)
			http.Error(w, "Failed to add user: ", http.StatusNotFound)
			return
		}
		user, loginOK, err := LogInCheckNotHashed(username, pwd)
		if loginOK {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		} else {
			fmt.Println("Failed to create user")
			http.Error(w, "Failed to Create user", http.StatusNotFound)
		}
		fmt.Println("User added with id: %v", id)

	case http.MethodDelete:
		fmt.Println("Delete request to users API")
		fmt.Println("This should be an attempt to remove a user account")
	case http.MethodPut:
		fmt.Println("Put request to users API")
		fmt.Println("Maybe this request could be used for changing passwords? But not i API so far")
	default:
		fmt.Println("Unsupportet request type to users API")
	}
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("sendHandler called")
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
	books, err := SearchBooksByTitleV1(requestData.Text)
	//fmt.Println(resp)
	var res string
	if err != nil {
		res = fmt.Sprintf("Error: %v\n", err)
	} else {
		res = fmt.Sprintf("Hits when searching for %v: %v\n", requestData.Text, books)
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
	fmt.Println("addBookHandler called")
	if r.Method != http.MethodPost {
		fmt.Println("Invalid request method ", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var book Book
	fmt.Println("boddy: ", r.Body)
	err := json.NewDecoder(r.Body).Decode(&book)
	fmt.Println("Book: ", book)
	for a, c := range r.Cookies() {
		fmt.Println(c, " | ", a)
	}

	if err != nil {
		fmt.Println("Failed to get cookie: ", err)
		http.Error(w, "Failed to get cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err != nil {
		fmt.Println("Invalid JSON", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//json.Unmarshal([]byte(r), &book)

	id, err := AddBook(book)
	if err != nil {
		fmt.Println("Failed to add book: ", err)
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
	fmt.Println("viewBooksBySellerHandler called")
	//sellerId := r.Header.Get("sellerid")
	user, err := getUserFromCookies(r)
	if err != nil {
		fmt.Println("Failed to get user: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	books, err := ViewSellerBooks(user.UserID)
	if err != nil {
		fmt.Println("Failed to get books: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var formattedBooks []map[string]interface{}

	for _, book := range books {
		fmt.Println("Price: ", book.Price)
		if !book.Price.Valid {
			book.Price = sql.NullInt32{0, true}
		}
		formattedBooks = append(formattedBooks, map[string]interface{}{
			"title":       book.Title,
			"sellerid":    book.SellerID,
			"description": book.Description.String,
			"price":       book.Price,
			"edition":     book.Edition.String,
			"stockAmount": book.StockAmount,
			"status":      book.Available,
		})
	}

	fmt.Printf("Books: %+v\n", formattedBooks)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"books":  formattedBooks,
	})
	if err != nil {
		fmt.Println("Failed to encode response: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func shoppingCartHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("shoppingCartHandler called")
	switch r.Method {
	case http.MethodGet:
		fmt.Println("Get request to shoppingcart API")
		fmt.Println("This should be an attempt to view the shopping cart")

		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		books, ids, err := GetShoppingChartBooks(user)
		if err != nil {
			fmt.Println("Failed to get cart: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var formattedBooks []map[string]interface{}
		i := 0
		for _, book := range books {
			fmt.Println("Price: ", book.Price)
			if !book.Price.Valid {
				book.Price = sql.NullInt32{Int32: 0, Valid: true}
			}
			formattedBooks = append(formattedBooks, map[string]interface{}{
				"title":       book.Title,
				"sellerid":    book.SellerID,
				"description": book.Description.String,
				"price":       book.Price,
				"edition":     book.Edition.String,
				"stockAmount": book.StockAmount,
				"status":      book.Available,
				"count":       ids[i],
			})
			i++
		}

		fmt.Printf("Books: %+v\n", formattedBooks)

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"books":  formattedBooks,
		})
		if err != nil {
			fmt.Println("Failed to encode response: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		fmt.Println("Post request to shoppingcart API")
		fmt.Println("This should be an attempt to add a book to the shopping cart")

		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		bookID := r.FormValue("bookID")
		count := r.FormValue("count")
		fmt.Println("bookID:%v, count:%v", bookID, count)
		bookIDint, err := strconv.Atoi(bookID)
		if err != nil {
			fmt.Println("Invalid bookID")
			http.Error(w, "Invalid bookID", http.StatusBadRequest)
			return
		}
		countInt, err := strconv.Atoi(count)
		if err != nil {
			fmt.Println("Invalid count")
			http.Error(w, "Invalid count", http.StatusBadRequest)
			return
		}
		newCount, err := AddBookToShoppingCart(user, int32(bookIDint), int32(countInt))
		if err != nil {
			fmt.Println("Failed to add to cart: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Book added to cart with count: %v", newCount, "former: ", count)
	case http.MethodDelete:
		fmt.Println("Delete request to shoppingcart API")
		fmt.Println("This should be an attempt to remove a book from the shopping cart")
		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		deleatAll := r.FormValue("deleateAll")
		if deleatAll == "True" {
			err = ResetShoppingCart(user)
			fmt.Println("Removed all book from cart")
		} else {
			bookID := r.FormValue("bookID")
			fmt.Println("bookID:%v", bookID)
			bookIDint, err := strconv.Atoi(bookID)
			if err != nil {
				fmt.Println("Invalid bookID")
				http.Error(w, "Invalid bookID", http.StatusBadRequest)
				return
			}
			err = SettCountInShoppingCart(user, int32(bookIDint), 0)
			if err != nil {
				fmt.Println("Failed to remove from cart: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Printf("Book removed from cart")
		}
	case http.MethodPut:
		fmt.Println("Put request to shoppingcart API")
		fmt.Println("This should be an attempt to change the count of a book in the shopping cart")
		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		bookID := r.FormValue("bookID")
		count := r.FormValue("count")
		fmt.Println("bookID:%v, count:%v", bookID, count)
		bookIDint, err := strconv.Atoi(bookID)
		if err != nil {
			fmt.Println("Invalid bookID")
			http.Error(w, "Invalid bookID", http.StatusBadRequest)
			return
		}
		countInt, err := strconv.Atoi(count)
		if err != nil {
			fmt.Println("Invalid count")
			http.Error(w, "Invalid count", http.StatusBadRequest)
			return
		}
		err = SettCountInShoppingCart(user, int32(bookIDint), int32(countInt))
		if err != nil {
			fmt.Println("Failed to set count in cart: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		fmt.Println("Unsupportet request type to shoppingcart API")
	}
}

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "AnkaAnka", //"AnkaAnka",
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
	http.HandleFunc("/API/users", userHandler)
	http.HandleFunc("/API/shoppingcart", shoppingCartHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
	fmt.Println("Server uppe!")
}

func getUserFromCookies(r *http.Request) (User, error) {
	IDcookie, err := r.Cookie("UserID")
	if err != nil {
		fmt.Println("Failed to get cookie: ", err)
		return User{}, err
	}
	userIDstr := IDcookie.Value
	if userIDstr == "" {
		fmt.Println("Missing userID")
		return User{}, fmt.Errorf("Missing userID")
	}
	userIDint, err := strconv.Atoi(userIDstr)
	if err != nil {
		fmt.Println("Invalid userID")
		return User{}, fmt.Errorf("Invalid userID")
	}
	user, err := GetUserByID(int32(userIDint))
	if err != nil {
		fmt.Println("Failed to get user: ", err)
		return User{}, err
	}
	userPsw, err := r.Cookie("Password")
	if err != nil {
		fmt.Println("Failed to get cookie: ", err)
		return User{}, err
	}
	user.Password = userPsw.Value
	username, err := r.Cookie("Username")
	if err != nil {
		fmt.Println("Failed to get cookie: ", err)
		return User{}, err
	}
	user.Username = username.Value

	return user, nil
}
