package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User: "root", //either set these in the command window,
		//or replace them with the appropriate user name and password here.
		//(I had trouble getting it to work with setting them in the command window)
		//(not sure how we want to do this in the end?)
		Passwd: "SnusmumrikenVolvo8041",
		//"AnkaAnka",
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

	// test add user and logincheck

	ids, error := AddUser("aSeller", "sellerPwd", sql.NullString{"KalleAnka@123.com", true})
	fmt.Println("added future seller user", ids, error)
	sellerid := ids
	ids, error = AddUser("aBuyer", "buyerPwd", sql.NullString{"KalleAnka@123.com", true})
	fmt.Println("added basic user", ids, error)
	//fmt.Println("kalle")
	user, _, errorr := LogInCheckNotHashed("aSeller", "sellerPwd")
	fmt.Println("user from login, seller user: ", user, err)
	user.Password = "sellerPwd"
	var userID = user.UserID
	fmt.Println("Login: ", userID, errorr)

	selleridCheck, error := AddSeller(user, "Testseller", sql.NullString{"", false})
	fmt.Println("addseller ", selleridCheck, "ought to be equal to", sellerid, error)

	user, err = GetUserByID(userID)
	//user.Password = "1234"
	fmt.Println("Get user: ", user, err)

	//sellerid = user.UserID

	//***********************************
	// test add book

	var b Book
	//fmt.Println(b)

	b2 := Book{Title: "title book 1", SellerID: sellerid}
	//fmt.Println(b2)

	b3 := Book{Title: "testbook2", SellerID: sellerid, Description: sql.NullString{"a nice long description", true}, Edition: sql.NullString{"edition 1", true}, StockAmount: 3, Available: false}
	//fmt.Println(b3)

	//id, error := AddBookMin("book 3 title", sellerid)
	//fmt.Println("addbook book 3", id, error)

	//b2.SellerID = sellerid
	//b3.SellerID = sellerid
	b.Title = "ERROR BOK"
	id, error := AddBook(b)
	fmt.Println("tried adding error book:", id, error)
	fmt.Println(b2.SellerID)
	id, error = AddBook(b2)
	fmt.Println("added b2:", id, error)

	id, error = AddBook(b3)
	fmt.Println("added b3:", id, error)

	books, err2 := GetBooksBySeller(1, true)

	fmt.Println("All books by seller ", sellerid, " independent of status")
	fmt.Println(sellerid, err2)
	DisplayBooklist(books)

	fmt.Println("All books by seller 1 that is for sale")
	books, err2 = GetBooksBySeller(1, false)
	fmt.Println(books, err2)
	DisplayBooklist(books)

	fmt.Println("All books whose title matches 'book', search version 1")
	books, err2 = SearchBooksByTitleV1("book")

	fmt.Println(books, err2)
	DisplayBooklist(books)

	fmt.Println("All books whose title matches 'book', search version 2")
	books, err2 = SearchBooksByTitleV2("book")

	fmt.Println(books, err2)
	DisplayBooklist(books)

	fmt.Println("All books whose title matches 'title book', search version 1")
	books, err2 = SearchBooksByTitleV1("title book")

	fmt.Println(books, err2)
	DisplayBooklist(books)

	fmt.Println("All books whose title matches 'title book', search version 2")
	books, err2 = SearchBooksByTitleV2("title book")

	fmt.Println(books, err2)
	DisplayBooklist(books)

}
