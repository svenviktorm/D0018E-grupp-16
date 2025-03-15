// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var loginpageURL string = "../start/login.html"
var startpageURL string = "../start.html"

// **** TYPE DEFINITIONS ****
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

/*
type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}
*/

//****** HTTP HANDLERS ******

func viewHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("viewHandler called")
	requestPath := r.URL.Path
	//fmt.Println(requestPath)
	//fmt.Println(r.Header)
	if requestPath == "/" {
		http.ServeFile(w, r, "website/start.html")
	} else {
		if strings.HasPrefix(requestPath, "/seller/") {
			fmt.Println("Seller only page, checking credentials")
			IDcookie, err := r.Cookie("UserID")
			fmt.Println(err, IDcookie)
			if err != nil || IDcookie.Value == "0" {
				fmt.Println("not a seller")
				http.Redirect(w, r, loginpageURL, http.StatusSeeOther)
				return
			} else {
				isSellerCookie, err := r.Cookie("IsSeller")
				if err != nil || isSellerCookie.Value != "true" {
					http.Error(w, "To access this page you must be registered as a seller", http.StatusForbidden)
					return
				}
			}
		} else if strings.HasPrefix(requestPath, "/admin/") {
			IDcookie, err := r.Cookie("UserID")
			if err != nil || IDcookie.Value == "0" {
				http.Redirect(w, r, loginpageURL, http.StatusSeeOther)
				return
			} else {
				isAdminCookie, err := r.Cookie("IsAdmin")
				if err != nil || isAdminCookie.Value != "true" {
					http.Error(w, "To access this page you must have administrator rights", http.StatusForbidden)
					return
				}
			}
		}

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
		uname := r.FormValue("username")
		pwd := r.FormValue("password")
		fmt.Printf("username:%v, password:%v, hash:%v", uname, pwd, hash(pwd))
		fmt.Println("")

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
		seller := r.FormValue("seller") == "seller"

		fmt.Println("username:%v, password:%v, mail:%v", username, pwd, email)
		emailSQL := sql.NullString{String: email, Valid: true}
		if email == "" {
			emailSQL = sql.NullString{String: "", Valid: false}
		}

		id, err := AddUser(username, pwd, emailSQL, seller)
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

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		fmt.Println("Invalid request method ", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	fmt.Println("logout request")
	IDcookie := http.Cookie{
		Name:   "UserID",
		Value:  "0", //A dummy value to overwrite the old just in case removal doesn't work for some reason (which it doesn't seemt to do)
		Path:   "/",
		MaxAge: 1, //Setting this to 0 SHOULD remove the cookie (according to internet), but that doesn't seem to work,
		// instead it just sets it to session? Setting it to 1 seem to make it disappear after a second has passed.
		// (in either case the dummy values work to ensure that the user credentials can't be used anymore)
		//HttpOnly: true,
	}
	http.SetCookie(w, &IDcookie)
	namecookie := http.Cookie{
		Name:   "Username",
		Value:  "", //A dummy value to overwrite the old just in case removal doesn't work for some reason
		Path:   "/",
		MaxAge: 1, //Setting this to 0 SHOULD remove the cookie (according to internet), but that doesn't seem to work?
		//HttpOnly: true,
	}
	http.SetCookie(w, &namecookie)
	pwdcookie := http.Cookie{
		Name:   "Password",
		Value:  "", //A dummy value to overwrite the old just in case removal doesn't work for some reason
		Path:   "/",
		MaxAge: 1, //Setting this to 0 SHOULD remove the cookie (according to internet), but that doesn't seem to work?
		//HttpOnly: true,
	}
	http.SetCookie(w, &pwdcookie)
	sellercookie := http.Cookie{
		Name:   "IsSeller",
		Value:  "false", //just in case removal doesn't work for some reason
		Path:   "/",
		MaxAge: 1, //Setting this to 0 SHOULD remove the cookie (according to internet), but that doesn't seem to work?
		//HttpOnly: true,
	}
	http.SetCookie(w, &sellercookie)
	admincookie := http.Cookie{
		Name:   "IsAdmin",
		Value:  "false", //just in case removal doesn't work for some reason
		Path:   "/",
		MaxAge: 1, //Setting this to 0 SHOULD remove the cookie (according to internet), but that doesn't seem to work?
		//HttpOnly: true,
	}
	http.SetCookie(w, &admincookie)
	http.Redirect(w, r, startpageURL, http.StatusSeeOther)
}

/*
	func sendHandler(w http.ResponseWriter, r *http.Request) {
		fmt.Println("sendHandler called")
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var requestData RequestData
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "error decoding JSON", http.StatusBadRequest)
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
*/
func bookHandler(w http.ResponseWriter, r *http.Request) {
	var books []Book
	var error error
	switch r.Method {
	case http.MethodGet:
		fmt.Println("Book search API called")
		fmt.Println(r)
		searchtype := r.FormValue("type")
		searchstring := r.FormValue("search")
		switch searchtype {
		case "Title":
			books, error = SearchBooksByTitle(searchstring, true)

		case "Author":
			books, error = SearchBooksByAuthor(searchstring, true)
		case "ISBN":
			isbn, err := strconv.Atoi(searchstring)
			if err != nil {
				fmt.Println("Something went wrong when converting ISBN to int")
				//TODO actuall error handling
			}
			books, error = SearchBooksByISBN(isbn, true)

		default:
			fmt.Println("Unimplemented search type")
		}
		if error != nil {
			fmt.Printf("some error: %v", error)
		}
		fmt.Println(books)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(books)
		if err != nil {
			fmt.Println("Failed to encode response: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		fmt.Println("Unsupportet request type to users API")
	}
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	fmt.Println("Raw request body:", string(bodyBytes))

	err = json.Unmarshal([]byte(bodyBytes), &book)
	if err != nil {
		fmt.Println("error decoding json")
		return
	}

	// get the userID cookie
	IDcookie, err := r.Cookie("UserID")
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println(book.ISBN)
	// convert cookie to integee
	var sellerId string = IDcookie.Value

	SellerIDint, err := strconv.Atoi(sellerId)
	fmt.Println(SellerIDint)
	book.SellerID = int32(SellerIDint)
	fmt.Println("Book: ", book.SellerID)

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
	fmt.Println(book)
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

	books, err := GetSellerBooks(user.UserID)
	if err != nil {
		fmt.Println("Failed to get books: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var formattedBooks []map[string]interface{}

	for _, book := range books {
		if !book.Price.Valid {
			book.Price = sql.NullInt32{0, true}
		}
		formattedBooks = append(formattedBooks, map[string]interface{}{
			"bookId":      book.BookID,
			"title":       book.Title,
			"sellerid":    book.SellerID,
			"author":      book.Author,
			"description": book.Description.String,
			"price":       book.Price,
			"edition":     book.Edition.String,
			"stockAmount": book.StockAmount,
			"available":   book.Available,
			"isbn":        book.ISBN,
		})
	}

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
				"bookid":      book.BookID,
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
		fmt.Println("type:", r.FormValue("type"))
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
	case http.MethodPut:
		fmt.Println("Put request to shoppingcart API")
		fmt.Println("requestType:", r.FormValue("requestType"))
		switch r.FormValue("requestType") {
		case "put":

			fmt.Println("This should be an attempt to change the count of a book in the shopping cart")
			user, err := getUserFromCookies(r)
			if err != nil {
				fmt.Println("Failed to get user: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			bookID := r.FormValue("bookID")
			count := r.FormValue("count")
			fmt.Println("bookID:, count:", bookID, count)
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
		case "delete":
			fmt.Println("Delete request to shoppingcart API")
			fmt.Println("This should be an attempt to remove a book from the shopping cart")
			user, err := getUserFromCookies(r)
			if err != nil {
				fmt.Println("Failed to get user: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			deleatAll := r.FormValue("deleateAll")
			fmt.Println("delete all:", deleatAll, "|")
			if deleatAll == "True" {
				err = ResetShoppingCart(user)
				fmt.Println("Removed all book from cart")
			} else {
				bookID := r.FormValue("bookID")
				fmt.Println("bookID:", bookID)
				r.ParseForm()
				for key, value := range r.Form {
					fmt.Printf("key: %v, value: %v\n", key, value)
				}
				fmt.Println("body: ", r.Form)
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
		default:
			fmt.Println("Unsupportet request type to shoppingcart API")
		}

	default:
		fmt.Println("Unsupportet request type to shoppingcart API")
	}
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("orderHandler called")
	switch r.Method {
	case http.MethodGet:
		fmt.Println("Get request to order API")
		fmt.Println("This should be an attempt to view the shopping cart")
		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("User: ", user)
	case http.MethodPost:
		switch r.FormValue("requestType") {
		case "createOrder":
			fmt.Println("Post request to order API")
			fmt.Println("This should be an attempt to create an order into reserved")
			user, err := getUserFromCookies(r)
			if err != nil {
				fmt.Println("Failed to get user: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = MakeShoppingCartIntoOrderReserved(user)
			if err != nil {
				fmt.Println("Failed to create order: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		}
	default:
		fmt.Println("Unsupportet request type to order API")
	}
}

func changeEmailHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("changeEmailHandler called")
	switch r.Method {

	case http.MethodPost:
		email := r.FormValue("changeEmail")

		fmt.Println("email:", email)
		emailSQL := sql.NullString{email, true}
		if email == "" {
			emailSQL = sql.NullString{"", false}
		}
		IDcookie, err := r.Cookie("UserID")
		if err != nil {
			fmt.Println("error getting userID from cookie")
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		userID, err := strconv.Atoi(IDcookie.Value)
		if err != nil {
			fmt.Println("error converting userID to int")
			http.Error(w, "Invalid UserID", http.StatusBadRequest)
			return
		}

		updatedEmail, err := changeEmail(emailSQL, int32(userID))
		if err != nil {
			fmt.Println("error updating email:", err)
			return
		}
		fmt.Println(updatedEmail)

	default:
		fmt.Println("Unsupportet request type to users API")
	}
}

func changeToSellerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("changeToSellerHandler called")

	cookies := r.Cookies()
	fmt.Println("All cookies:")
	for _, cookie := range cookies {
		fmt.Printf("Cookie Name: %s, Cookie Value: %s\n", cookie.Name, cookie.Value)
	}

	switch r.Method {

	case http.MethodPost:

		//err := r.ParseForm()
		//if err != nil {
		//	http.Error(w, "Error parsing form", http.StatusBadRequest)
		//	return
		//}
		user, err := getUserFromCookies(r)
		if err != nil {
			fmt.Println("Failed to get user: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		name := r.FormValue("name")
		description := r.FormValue("description")
		fmt.Println("name:", name)
		fmt.Println("description:", description)

		changedSeller, err := changeToSeller(int32(user.UserID), user.Username, user.Password, user.Email, description, name)
		fmt.Println("changeToSeller called", description)
		fmt.Println("körs ens denna")
		if err != nil {
			fmt.Println("error changing to seller:", err)
			return
		}
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "Seller status updated successfully"})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		}
		fmt.Println("changed to seller: ", changedSeller)

	default:
		fmt.Println("Unsupportet request type to users API")
	}
}

func editBookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("editBookHandler called")
	if r.Method != http.MethodPost {
		fmt.Println("Invalid request method ", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var book Book
	fmt.Println("body: ", r.Body)

	err := json.NewDecoder(r.Body).Decode(&book)
	if err != nil {
		fmt.Println("error decoding json:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Decoded Book: %+v\n", book)

	IDcookie, err := r.Cookie("UserID")
	if err != nil {
		fmt.Println("Failed to get cookie: ", err)
		http.Error(w, "Failed to get cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// convert cookie to integer
	sellerId := IDcookie.Value
	SellerIDint, err := strconv.Atoi(sellerId)
	if err != nil {
		fmt.Println("Failed to convert cookie to integer:", err)
		http.Error(w, "Invalid seller ID", http.StatusBadRequest)
		return
	}

	book.SellerID = int32(SellerIDint)
	fmt.Println("Book SellerID:", book.SellerID)

	id, err := editBook(book)
	if err != nil {
		fmt.Println("Failed to edit book: ", err)
		http.Error(w, "Failed to edit book: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Book edited successfully",
		"id":      id,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		fmt.Println("Error encoding JSON response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Println("JSON response sent:", response)
}

func removeBookHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Available bool  `json:"available"`
		BookId    int32 `json:"bookId"`
	}
	fmt.Println("availaible: ", data.Available, "bookid: ", data.BookId)
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding json", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	err = removeBook(data.Available, data.BookId)
	if err != nil {
		http.Error(w, "Error updating book availability", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func viewBooksHandler(w http.ResponseWriter, r *http.Request) {
	books, err := viewBooks()
	if err != nil {
		fmt.Println("Failed to get books: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var formattedBooks []map[string]interface{}

	for _, book := range books {
		if !book.Price.Valid {
			book.Price = sql.NullInt32{0, true}
		}
		formattedBooks = append(formattedBooks, map[string]interface{}{
			"bookId":      book.BookID,
			"title":       book.Title,
			"sellerid":    book.SellerID,
			"author":      book.Author,
			"description": book.Description.String,
			"price":       book.Price,
			"edition":     book.Edition.String,
			"stockAmount": book.StockAmount,
			"available":   book.Available,
			"isbn":        book.ISBN,
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

// *** Variables ***
var db *sql.DB

// **** MAIN ****

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
	http.HandleFunc("OPTIONS /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-control-allow-methods", "POST, GET, DELETE")
	})
	http.HandleFunc("/", viewHandler)
	http.HandleFunc("/add_book", addBookHandler)
	http.HandleFunc("/viewSellerBook", viewBooksBySellerHandler)
	http.HandleFunc("/email", changeEmailHandler)
	http.HandleFunc("/changeToSeller", changeToSellerHandler)
	http.HandleFunc("/edit_book", editBookHandler)
	http.HandleFunc("/remove_book", removeBookHandler)
	http.HandleFunc("/viewBooks", viewBooksHandler)
	http.HandleFunc("/orders", viewBooksHandler)
	//http.HandleFunc("POST /", viewHandler)
	fmt.Println("a!")
	http.HandleFunc("/root/", rootHandler)
	fmt.Println("b!")
	//http.HandleFunc("/send", sendHandler)
	//fmt.Println("c!")
	http.HandleFunc("/API/users", userHandler)

	http.HandleFunc("/API/sessions", sessionHandler)

	http.HandleFunc("/API/shoppingcart", shoppingCartHandler)
	http.HandleFunc("/API/books", bookHandler)

	log.Fatal(http.ListenAndServe(":80", nil))
	fmt.Println("Server uppe!")
}

// checkar inte om lösenordet är rätt
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
		fmt.Println("Failed to get password: ", err)
		return User{}, err
	}
	user.Password = userPsw.Value

	return user, nil
}
