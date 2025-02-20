package main

import (
	"database/sql"
	"fmt"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
)

type User struct {
	UserID   int32
	Username string
	Password string
	Email    sql.NullString
	IsAdmin  bool
	IsSeller bool
}

type Seller struct {
	SellerID	int32
	Name		string
	Description sql.NullString
}

type Book struct {
	BookID		int32
	Title       string			`json:"title"`
	SellerID    int32			`json:"sellerid"`
	Edition     sql.NullString	`json:"edition"`
	Description sql.NullString	`json:"description"`
	StockAmount int32  			`json:"stockAmount"`//since the 'zero value' of int is 0 the value of StockAmount will be 0 if not set explicitly, which works fine in this case. So no need for a Null-type.
	Available	bool 			`json:"status"`//This will have the value false if not set, not sure if that is what we want or not? Status feels like something that should be set internally rather than directly by the seller(?) so might be no need to have a good automatic default?
	ISBN		sql.NullInt32
	NumRatings  sql.NullInt32 
	SumRatings 	sql.NullInt32
	Price 		int32	`json:"price"`
}
func hash(plaintext string) (int64) {
	h := sha3.New256()
	h.Write([]byte(plaintext))
	hash := h.Sum(nil)
	//first 64 bits
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func AddUser(username string, password string, email sql.NullString) (int32, error) {
	fmt.Println("kalle")
	var passwordHash int64 = hash(password)
	result, err := db.Exec("INSERT INTO Users (username, PasswordHash, email, IsAdmin, IsSeller) VALUES (?, ?, ?, ? , ?)", username, passwordHash, email,false,false)
	if err != nil {
		fmt.Println("anka2")
		return 0, fmt.Errorf("AddUser: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("anka1")
		return 0, fmt.Errorf("AddUser: %v", err)
	}
	var i32 int32 = int32(id)
	fmt.Println("anka")
	return i32, nil
}
//for checking if a username and password exists
func LogInCheckNotHashed(username string, password string) (user User, loginSucces bool, err error) {
	var passwordHash int64 = hash(password)
	return LoginCheck(username, passwordHash)
}
//for checking a username with an already hashed password
func LoginCheck(username string, passwordHash int64) (user User, loginSucces bool, err error) {
	
	rows, err := db.Query("SELECT Id, Email IsAdmin, IsSeller FROM Users WHERE Username = ? AND PasswordHash = ? ",username,passwordHash)
	if err != nil {
		var user User = User{}
		return user , false, fmt.Errorf("LoginCheck: %v", err)
	}
	
	for rows.Next() {
		var id int32
		var isAdmin bool
		var isSeller bool
		var email sql.NullString
		err := rows.Scan(&id,&email, &isAdmin, &isSeller)
		if err != nil {
			fmt.Errorf("LoginCheck: %v", err)
		}
		var user User = User{id, username, "", email, isAdmin, isSeller}
		return user, true, err
	}
	if err != nil {
		return User{}, false, fmt.Errorf("LoginCheck: No User found %v", err)
	}
	return User{},false, fmt.Errorf("LoginCheck: No User found")
}

func AddSeller(user User,name string, description sql.NullString) (int32, error) {
	//check if user exists can be converted to use userid as input instead
	user,loginSucces, loginerr := LogInCheckNotHashed(user.Username, user.Password ) 
	if loginerr != nil {
		return -1, fmt.Errorf("AddSeller: %v", loginerr)
	}

	if !loginSucces {
		return -1, fmt.Errorf("AddSeller: loginsfail %v", loginerr)
	}

	fmt.Println("ANKA; ",user.UserID,loginerr)
	
	tx, dberr := db.Begin()
	//defer db.Close()
	if dberr != nil {
		return -2, fmt.Errorf("transaction erroor:",dberr)
	} 	
	result, err := db.Exec("INSERT INTO Sellers (Name, Id, Description) VALUES (?, ?, ?)", name, user.UserID, description)
	if err != nil {
		tx.Rollback()
		return -3, fmt.Errorf("AddSeller: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return -4, fmt.Errorf("AddSeller: %v", err)
	}
	db.Exec("UPDATE users SET IsSeller = True WHERE ID = ?",user.UserID)
	if err != nil {
		tx.Rollback()
		return -5, fmt.Errorf("AddSeller: %v", err)
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("Error committing transaction:", err)
	}
	return int32(id), nil 
}
//retursn a user struct from their userID
func GetUserByID(userID int32) (User, error) {
	rows, err := db.Query("SELECT Id, Username, PasswordHash, Email, IsAdmin, IsSeller FROM Users WHERE Id = ? ",userID)
	if err != nil {
		fmt.Errorf("Error getting:", err)
	}
	for rows.Next(){
		var user User 
		if err := rows.Scan(&user.UserID,&user.Username,&user.Password, &user.Email, &user.IsAdmin, &user.IsSeller);  err != nil {
			return User{}, fmt.Errorf("GetUserID %q: %v", userID, err)
		}
		// the returend password will be hashed and therefore usless and therefore removed to not cause confusion
		user.Password = ""
		//there should only be one user per ID otherwise something is wrong with the database 
		return user, nil
	}
	return User{}, nil
}

func GetBooksBySeller(sellerID int, includeAvailable bool) ([]Book, error) {

	var books []Book
	var err error
	var rows *sql.Rows

	if includeAvailable {
		rows, err = db.Query("SELECT Id,Title,Edition,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE SellerID = ?", sellerID)

	} else {
		rows, err = db.Query("SELECT Id,Title,Edition,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE SellerID = ? AND Available=TRUE", sellerID)
	}

	if err != nil {
		return nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		if err := rows.Scan( &b.BookID , &b.Title, &b.Edition, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
			return nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
	}
	return books, nil
}
//creates a book from minimal information
func AddBookMin(title string, sellerID int32) (int32, error) {
	nullStr := sql.NullString{
		Valid: false,
	}
	nullInt32 := sql.NullInt32{
		Valid: false,
	}

	zeroInt32 := sql.NullInt32{
		Valid: false,
		Int32: 0,
	}
	//id of -99 should not be used
	var book = Book{-99,title, sellerID, nullStr, nullStr, 0, false, nullInt32, zeroInt32, zeroInt32, 0}
	return AddBook(book)

}
// will not use the id of the book but create one
func AddBook(book Book) (int32, error) {
	user, err := GetUserByID(book.SellerID)
	if err != nil {
		return -1, fmt.Errorf("AddSeller: %v", err)
	}
	//check if seller exists can be optimized
	user, loginSucces ,  loginerr := LogInCheckNotHashed(user.Username, user.Password ) 
	if loginerr != nil  {
		return -1, fmt.Errorf("AddSeller: %v", loginerr)
	}

	if !loginSucces {
		return -1, fmt.Errorf("AddSeller: loginsfail %v", loginerr)
	}

	result, err := db.Exec("INSERT INTO Books (Title, SellerID, Edition, Description, StockAmount, Available, ISBN, NumRatings, SumRatings, Price) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", book.Title, user.UserID, book.Edition, book.Description, book.StockAmount, book.Available, book.ISBN, 0, 0, book.Price)
	if err != nil {
		return -1, fmt.Errorf("addBook: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -2, fmt.Errorf("addBook: %v", err)
	}
	return int32(id), nil

}

func SearchBooksByTitleV1(titlesearch string) ([]Book, error) {
	var books []Book
	var ids []int32
	var err error
	var rows *sql.Rows

	rows, err = db.Query("SELECT Id,Title,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE MATCH(Title) AGAINST(?)", titlesearch)

	if err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev1 %q: %v", titlesearch, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		var i int32
		if err := rows.Scan(&b.BookID ,&b.Title, &b.Edition,&b.Description, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
			return nil, fmt.Errorf("searchBooksByTitlev1 %q: %v", titlesearch, err)
		}
		books = append(books, b)
		ids = append(ids, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev1 %q: %v", titlesearch, err)
	}
	return books, nil
}

func SearchBooksByTitleV2(titlesearch string) ([]Book, error) {
	var books []Book
	var err error
	var rows *sql.Rows

	titlesearch = "%" + titlesearch + "%"
	rows, err = db.Query("SELECT Id,Title,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE Title LIKE ?", titlesearch)

	if err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev2 %q: %v", titlesearch, err) //TODO fix format
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.BookID ,&b.Title, &b.Edition,&b.Description, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
			return nil, fmt.Errorf("searchBooksByTitlev2 %q: %v", titlesearch, err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev2 %q: %v", titlesearch, err)
	}
	return books, nil
}

func viewSellerBooks(sellerID int) ([]Book, error) {
	var books []Book

	rows, err := db.Query("SELECT Title, Description, Price, Edition, StockAmount, Available, ISBN, NumRatings, SumRatings FROM Books WHERE SellerID = ?", sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var book Book
		err := rows.Scan(&book.Title, &book.Description, &book.Price, &book.Edition, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings)
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
	fmt.Println("| Ttitle | Edition | stock amount | seller name | ")
	for _, b := range books {
		if b.Edition.Valid {
			edition = b.Edition.String
		} else {
			edition = "NULL"
		}
		fmt.Println("|", b.Title, "|", edition, "|", b.StockAmount, "|",)

	}
}
