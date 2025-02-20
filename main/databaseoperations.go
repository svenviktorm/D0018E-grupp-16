package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type Book struct {
	Title       string         `json:"title"`
	SellerID    int            `json:"sellerid"`
	Description sql.NullString `json:"description"`
	Price       int            `json:"price"`
	Edition     sql.NullString `json:"edition"`
	Cathegory   string         `json:"cathegory"`
	StockAmount int            `json:"stockAmount"` //since the 'zero value' of int is 0 the value of StockAmount will be 0 if not set explicitly, which works fine in this case. So no need for a Null-type.
	Status      bool           `json:"status"`      //This will have the value false if not set, not sure if that is what we want or not? Status feels like something that should be set internally rather than directly by the seller(?) so might be no need to have a good automatic default?
}

func AddSeller(name string) (int64, error) {
	result, err := db.Exec("INSERT INTO Sellers (Name) VALUES (?)", name)
	if err != nil {
		return 0, fmt.Errorf("AddSeller: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("AddSeller: %v", err)
	}
	return id, nil
}

func GetBooksBySeller(sellerID int, includeAll bool) ([]Book, []int, error) {

	var books []Book
	var ids []int
	var err error
	var rows *sql.Rows

	if includeAll {
		rows, err = db.Query("SELECT Id,Edition,StockAmount FROM Books WHERE SellerID = ?", sellerID)

	} else {
		rows, err = db.Query("SELECT Id,Edition,StockAmount FROM Books WHERE SellerID = ? AND Status=TRUE", sellerID)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		var i int
		if err := rows.Scan(&i, &b.Title, &b.Edition, &b.StockAmount); err != nil {
			return nil, nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
		}
		books = append(books, b)
		ids = append(ids, i)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
	}
	return books, ids, nil
}

func AddBookMin(title string, sellerID int) (int64, error) {
	nullStr := sql.NullString{
		Valid: false,
	}
	var book = Book{title, sellerID, nullStr, 0, nullStr, "", 0, false}
	return AddBook(book)

}

func AddBook(book Book) (int64, error) {

	// Convert to JSON to check output
	jsonData, _ := json.MarshalIndent(book, "", "  ")
	fmt.Println(string(jsonData))
	sellerid := 1
	result, err := db.Exec("INSERT INTO Books (Title, SellerID, Description, Price, Edition, Cathegory, StockAmount, Status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", book.Title, sellerid, book.Description, book.Price, book.Edition, book.Cathegory, book.StockAmount, book.Status)
	if err != nil {
		return 0, fmt.Errorf("addBook: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addBook: %v", err)
	}
	return id, nil
}

func SearchBooksByTitleV1(titlesearch string) ([]Book, []int, error) {
	var books []Book
	var ids []int
	var err error
	var rows *sql.Rows

	rows, err = db.Query("SELECT Id,Title,Edition,StockAmount FROM Books WHERE MATCH(Title) AGAINST(?)", titlesearch)

	if err != nil {
		return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		var i int
		if err := rows.Scan(&i, &b.Title, &b.Edition, &b.StockAmount); err != nil {
			return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err)
		}
		books = append(books, b)
		ids = append(ids, i)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err)
	}
	return books, ids, nil
}

func SearchBooksByTitleV2(titlesearch string) ([]Book, []int, error) {
	var books []Book
	var ids []int
	var err error
	var rows *sql.Rows

	titlesearch = "%" + titlesearch + "%"
	rows, err = db.Query("SELECT Id,Title,Edition,StockAmount FROM Books WHERE Title LIKE ?", titlesearch)

	if err != nil {
		return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		var i int
		if err := rows.Scan(&i, &b.Title, &b.Edition, &b.StockAmount); err != nil {
			return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err)
		}
		books = append(books, b)
		ids = append(ids, i)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("searchBooksByTitle %q: %v", titlesearch, err)
	}
	return books, ids, nil
}

func viewSellerBooks(sellerID int) ([]Book, error) {
	var books []Book

	rows, err := db.Query("SELECT Title, Description, Price, Edition, Cathegory, StockAmount FROM Books WHERE SellerID = ?", sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var book Book
		err := rows.Scan(&book.Title, &book.Description, &book.Price, &book.Edition, &book.Cathegory, &book.StockAmount)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

func DisplayBooklist(books []Book) {
	// just for testing purposes
	var edition string
	for _, b := range books {
		if b.Edition.Valid {
			edition = b.Edition.String
		} else {
			edition = "NULL"
		}
		fmt.Println("|", b.Title, "|", "|", edition, "|", b.StockAmount, "|")

	}
}
