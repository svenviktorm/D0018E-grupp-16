package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"

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
	SellerID    int32
	Name        string
	Description sql.NullString
}

type Book struct {
	BookID   int32  `json:"bookId"`
	Title    string `json:"title"`
	SellerID int32  `json:"sellerId"`

	Author string `json:"author"`

	Edition     sql.NullString `json:"edition"`
	Description sql.NullString `json:"description"`
	StockAmount int32          `json:"stockAmount"` //since the 'zero value' of int is 0 the value of StockAmount will be 0 if not set explicitly, which works fine in this case. So no need for a Null-type.
	Available   bool           `json:"available"`   //This will have the value false if not set, not sure if that is what we want or not? Status feels like something that should be set internally rather than directly by the seller(?) so might be no need to have a good automatic default?
	ISBN        sql.NullInt64  `json:"isbn"`
	NumRatings  sql.NullInt32  `json:"numratings"`
	SumRatings  sql.NullInt32  `json:"sumratings"`
	Price       sql.NullInt32  `json:"price"`
}

type BookReview struct {
	Id     int32          `json:"id"`
	BookID int32          `json:"bookid"`
	UserID int32          `json:"userid"`
	Text   sql.NullString `json:"text"`
	Rating int            `json:"rating"`
}

func hash(plaintext string) int64 {
	h := sha3.New256()
	h.Write([]byte(plaintext))
	hash := h.Sum(nil)
	//first 64 bits
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func AddUser(username string, password string, email sql.NullString, isSeller bool) (int32, error) {
	fmt.Println("kalle")
	var passwordHash int64 = hash(password)
	result, err := db.Exec("INSERT INTO Users (username, PasswordHash, email, IsAdmin, IsSeller) VALUES (?, ?, ?, ? , ?)", username, passwordHash, email, false, isSeller)
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

	if isSeller == true {
		newUser := User{
			UserID:   i32,
			Username: username,
			Password: password,
			Email:    email,
			IsSeller: isSeller,
			IsAdmin:  false,
		}
		_, err := AddSeller(newUser, username, sql.NullString{String: "", Valid: false})
		if err != nil {
			fmt.Println("Error adding seller:", err)
			return 0, fmt.Errorf("AddSeller: %v", err)
		}
	}
	fmt.Println("anka")
	return i32, nil
}

// for checking if a username and password exists
func LogInCheckNotHashed(username string, password string) (user User, loginSucces bool, err error) {
	var passwordHash int64 = hash(password)
	return LoginCheck(username, passwordHash)
}

// for checking a username with an already hashed password
func LoginCheck(username string, passwordHash int64) (user User, loginSucces bool, err error) {

	rows, err := db.Query("SELECT Id, Email, IsAdmin, IsSeller FROM Users WHERE Username = ? AND PasswordHash = ? ", username, passwordHash)
	if err != nil {
		var user User = User{}
		return user, false, fmt.Errorf("LoginCheck: Queary: %v", err)
	}

	for rows.Next() {
		var id int32
		var isAdmin bool
		var isSeller bool
		var email sql.NullString
		err := rows.Scan(&id, &email, &isAdmin, &isSeller)
		if err != nil {
			return User{}, false, fmt.Errorf("LoginCheck: Scan: %v", err)
		}
		fmt.Println("id: ", id)
		var user User = User{id, username, "", email, isAdmin, isSeller}
		return user, true, err
	}
	return User{}, false, nil
}

func AddSeller(user User, name string, description sql.NullString) (int32, error) {
	//check if user exists can be converted to use userid as input instead
	user, loginSucces, loginerr := LogInCheckNotHashed(user.Username, user.Password)
	if loginerr != nil {
		return -1, fmt.Errorf("AddSeller: %v", loginerr)
	}

	if !loginSucces {
		return -1, fmt.Errorf("AddSeller: loginsfail %v", loginerr)
	}

	fmt.Println("ANKA; ", user.UserID, loginerr)

	tx, dberr := db.Begin()
	//defer db.Close()
	if dberr != nil {
		return -2, fmt.Errorf("transaction error:", dberr)
	}

	var descriptionValue interface{}
	if description.Valid {
		descriptionValue = description.String
	} else {
		descriptionValue = nil
	}

	result, err := db.Exec("INSERT INTO Sellers (Name, Id, Description) VALUES (?, ?, ?)", name, user.UserID, descriptionValue)

	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -3, fmt.Errorf("AddSeller: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -4, fmt.Errorf("AddSeller: %v", err)
	}
	db.Exec("UPDATE Users SET IsSeller = True WHERE ID = ?", user.UserID)
	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -5, fmt.Errorf("AddSeller: %v", err)
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("Error committing transaction:", err)
	}
	return int32(id), nil
}

// retursn a user struct from their userID
func GetUserByID(userID int32) (User, error) {
	rows, err := db.Query("SELECT Id, Username, PasswordHash, Email, IsAdmin, IsSeller FROM Users WHERE Id = ? ", userID)
	if err != nil {
		fmt.Errorf("Error getting:", err)
	}
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.Username, &user.Password, &user.Email, &user.IsAdmin, &user.IsSeller); err != nil {
			return User{}, fmt.Errorf("GetUserID %q: %v", userID, err)
		}
		// the returend password will be hashed and therefore usless and therefore removed to not cause confusion
		user.Password = ""
		//there should only be one user per ID otherwise something is wrong with the database
		return user, nil
	}
	return User{}, nil
}

/*
//There where two functions that did this
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
		if err := rows.Scan(&b.BookID, &b.Title, &b.Edition, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
			return nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getBooksBySeller %q: %v", sellerID, err)
	}
	return books, nil
}
*/
/*
// creates a book from minimal information
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
	var book = Book{-99, title, sellerID, nullStr, nullStr, 0, true, nullInt32, zeroInt32, zeroInt32, nullInt32}
	return AddBook(book)

}
*/

// will not use the id of the book but create one
func AddBook(book Book) (int32, error) {
	user, err := GetUserByID(book.SellerID)
	if err != nil {
		return -1, fmt.Errorf("Addbook: %v", err)
	}
	//check if seller exists can be optimized
	//user, loginSucces ,  loginerr := LogInCheckNotHashed(user.Username, user.Password )
	/*if loginerr != nil  {
		return -1, fmt.Errorf("Addbook: %v", loginerr)
	}

	if !loginSucces {
		return -1, fmt.Errorf("Addbook: loginsfail %v", loginerr)
	}*/
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println(book.ISBN)
	result, err := db.Exec("INSERT INTO Books (Title, Author, SellerID, Edition, Description, StockAmount, Available, ISBN, NumRatings, SumRatings, Price) VALUES (?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?)", book.Title, book.Author, user.UserID, book.Edition, book.Description, book.StockAmount, book.Available, book.ISBN, 0, 0, book.Price)
	if err != nil {
		return -1, fmt.Errorf("addBook: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -2, fmt.Errorf("addBook: %v", err)
	}
	return int32(id), nil

}

func changeEmail(email sql.NullString, id int32) (sql.Result, error) {
	result, err := db.Exec("UPDATE Users SET Email = ? WHERE Id = ?", email, id)
	if err != nil {
		fmt.Println("error updating email")
		return result, err
	}
	return result, nil
}

func changeToSeller(id int32, username string, password string, email sql.NullString, description string, name string) (int32, error) {
	db.Exec("UPDATE Users SET IsSeller = ? WHERE Id = ?", true, id)
	newUser := User{
		UserID:   id,
		Username: username,
		Password: password,
		Email:    email,
		IsSeller: true,
		IsAdmin:  false,
	}
	sellerid, err := AddSeller(newUser, name, sql.NullString{String: description, Valid: true})
	if err != nil {
		fmt.Println("Error adding seller:", err)
		return 0, fmt.Errorf("AddSeller: %v", err)
	}
	if err != nil {
		fmt.Println("error updating email")
		return id, err
	}
	return sellerid, nil
}

func editBook(book Book) (int32, error) {
	result, err := db.Exec("UPDATE Books SET Title = ?, Description = ?, Price = ?, Edition = ?, StockAmount = ?, ISBN = ? WHERE Id = ?", book.Title, book.Description, book.Price, book.Edition, book.StockAmount, book.ISBN, book.BookID)
	if err != nil {
		return -1, fmt.Errorf("addBook: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -2, fmt.Errorf("addBook: %v", err)
	}

	return int32(id), nil
}

func createReview(userId int32, bookId int32, text string, rating int) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM BookReviews WHERE UserID = ? AND BookID = ?", userId, bookId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing review: %v", err)
	}
	if count > 0 {
		return fmt.Errorf("user has already reviewed this book")
	}

	var sellerId int32
	err = db.QueryRow("SELECT SellerID FROM Books WHERE Id = ?", bookId).Scan(&sellerId)
	if err != nil {
		return fmt.Errorf("failed to check book seller: %v", err)
	}
	if sellerId == userId {
		return fmt.Errorf("sellers cannot review their own books")
	}

	_, err = db.Exec("INSERT INTO BookReviews (BookID, UserID, Text, Rating) VALUES (?, ?, ?, ?)", bookId, userId, text, rating)
	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}

	var numRatings int
	var sumRatings int

	err = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(Rating), 0) FROM BookReviews WHERE BookID = ?", bookId).Scan(&numRatings, &sumRatings)
	if err != nil {
		return fmt.Errorf("failed to fetch updated ratings: %v", err)
	}

	averageRating := sumRatings / numRatings

	_, err = db.Exec("UPDATE Books SET NumRatings = ?, SumRatings = ? WHERE Id = ?", numRatings, averageRating, bookId)
	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}

	return nil
}

func getReviews(bookId int32) ([]BookReview, int, error) {
	fmt.Println("getReviews called", bookId)

	var sumRatings int
	err := db.QueryRow("SELECT SumRatings FROM Books WHERE Id = ?", bookId).Scan(&sumRatings)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get sumRatings: %v", err)
	}

	rows, err := db.Query("SELECT Id, BookID, UserID, Text, Rating FROM BookReviews WHERE BookID = ?", bookId)
	if err != nil {
		return nil, 0, fmt.Errorf("getReview1: %v", err)
	}
	defer rows.Close()
	var reviews []BookReview
	for rows.Next() {
		var bookReview BookReview
		err := rows.Scan(&bookReview.Id, &bookReview.BookID, &bookReview.UserID, &bookReview.Text, &bookReview.Rating)
		if err != nil {
			return nil, 0, fmt.Errorf("getBookById2: %v", err)
		}
		reviews = append(reviews, bookReview)
		fmt.Println("bookreview: ", bookReview)
	}

	return reviews, sumRatings, nil
}

/*

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
		if err := rows.Scan(&b.BookID, &b.Title, &b.Edition, &b.Description, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
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
*/

func removeBook(available bool, bookId int32) error {
	var exists bool

	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM Books WHERE Id = ?)", bookId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking book existence: %v", err)
	}

	if !exists {
		return fmt.Errorf("book with ID %d does not exist", bookId)
	}
	db.Exec("UPDATE Books SET Available = ? WHERE Id = ?", available, bookId)
	return nil
}

func SearchBooksByTitle(titlesearch string, onlyAvailable bool) ([]Book, error) {

	var err error
	var rows *sql.Rows

	titlesearch = "%" + titlesearch + "%"
	if onlyAvailable {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE Available AND Title LIKE ?", titlesearch)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE Title LIKE ?", titlesearch)

	}

	if err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev2 %v: %v", titlesearch, err)
	}
	return extractBooksFromSQLresult(rows)
}

func SearchBooksByAuthor(authorsearch string, onlyAvailable bool) ([]Book, error) {

	var err error
	var rows *sql.Rows

	if onlyAvailable {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE Available AND MATCH(Author) AGAINST(?)", authorsearch)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE MATCH(Author) AGAINST(?)", authorsearch)

	}

	if err != nil {
		return nil, fmt.Errorf("searchBooksByAuthor %v: %v", authorsearch, err)
	}

	return extractBooksFromSQLresult(rows)

}

func SearchBooksByISBN(isbn int, onlyAvailable bool) ([]Book, error) {
	var err error
	var rows *sql.Rows

	if onlyAvailable {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE Available AND ISBN=?", isbn)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price FROM Books WHERE ISBN=?", isbn)

	}

	if err != nil {
		return nil, fmt.Errorf("searchBooksByTitlev2 %v: %v", isbn, err)
	}
	return extractBooksFromSQLresult(rows)
}

func extractBooksFromSQLresult(rows *sql.Rows) ([]Book, error) {
	var books []Book
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.BookID, &b.Title, &b.Author, &b.SellerID, &b.Edition, &b.Description, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price); err != nil {
			return nil, fmt.Errorf("searchBooks: %v", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("searchBooks: %v", err)
	}
	return books, nil
}

func GetSellerBooks(sellerID int32) ([]Book, error) {
	//Changed the name: this function doesn't view the books, it just returns a list of books, so the name ViewSellerBooks was misleading.
	var books []Book

	rows, err := db.Query("SELECT Id, Title, Author, Description, Price, Edition, StockAmount, Available, ISBN, NumRatings, SumRatings, SellerID FROM Books WHERE SellerID = ?", sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var book Book
		err := rows.Scan(&book.BookID, &book.Title, &book.Author, &book.Description, &book.Price, &book.Edition, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings, &book.SellerID)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

// I think this isn't used anymore?
//func viewBooks() ([]Book, error) {

//	var books []Book

//	rows, err := db.Query("SELECT Id, Title, Description, Price, Edition, StockAmount, Available, ISBN, NumRatings, SumRatings, SellerID FROM Books")
//	if err != nil {
//		return nil, err
//	}
//	defer rows.Close()

//	for rows.Next() {
//		var book Book
//		err := rows.Scan(&book.BookID, &book.Title, &book.Description, &book.Price, &book.Edition, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings, &book.SellerID)
//		if err != nil {
//			return nil, err
//		}
//		books = append(books, book)
//	}
//	return books, nil
//}

func AddBookToShoppingCart(user User, bookID int32, count int32) (newCount int32, err error) {
	user, successLogin, err := LogInCheckNotHashed(user.Username, user.Password)
	if err != nil || !successLogin {
		return -1, fmt.Errorf("Invalid User: %v", err)
	}
	rows, err := db.Query("SELECT Quantity FROM InShoppingCart WHERE UserID = ? AND BookID = ?", user.UserID, bookID)
	if err != nil {
		return -1, fmt.Errorf("AddBookToShoppingCart: %v", err)
	}
	var quantity int32 = 0
	// first has double meaning as either the first insert of this column of check if there are multiple columns
	first := true
	for rows.Next() {
		if first {
			err := rows.Scan(&quantity)
			if err != nil {
				return -quantity, fmt.Errorf("AddBookToShoppingCart1: %v", err)
			}
			first = false
		} else {
			return -quantity, fmt.Errorf("AddBookToShoppingCart: More than one row returned")
		}
	}
	if first {
		_, err := db.Exec("INSERT INTO InShoppingCart (UserID, BookID, Quantity) VALUES (?, ?, ?)", user.UserID, bookID, count)
		if err != nil {
			return count, fmt.Errorf("AddBookToShoppingCart2: %v", err)
		}
		return count, nil
	} else {
		_, err := db.Exec("UPDATE InShoppingCart SET Quantity = ? WHERE UserID = ? AND BookID = ?", quantity+count, user.UserID, bookID)
		if err != nil {
			return quantity + count, fmt.Errorf("AddBookToShoppingCart3: %v", err)
		}
		return quantity + count, nil
	}
}

// can be used to remove a book from the shopping cart
// if count is set to 0 the book will be removed from the shopping cart
func SettCountInShoppingCart(user User, bookID int32, count int32) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("Invalid User/login invalid: %v", err)
	}
	if count != 0 {
		_, err = db.Exec("UPDATE InShoppingCart SET Quantity = ? WHERE UserID = ? AND BookID = ?", count, user.UserID, bookID)
		if err != nil {
			return fmt.Errorf("SettCountInShoppingCart1: %v", err)
		}
		return nil
	} else {
		_, err = db.Exec("DELETE FROM InShoppingCart WHERE UserID = ? AND BookID = ?", user.UserID, bookID)
		if err != nil {
			return fmt.Errorf("SettCountInShoppingCart2: %v", err)
		}
		return nil
	}
}

func ResetShoppingCart(user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("Invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("DELETE FROM InShoppingCart WHERE UserID = ? ", user.UserID)
	return err
}

func GetShoppingChartBooks(user User) ([]Book, []int32, error) {
	user, successLogin, err := LogInCheckNotHashed(user.Username, user.Password)
	if err != nil || !successLogin {
		return nil, nil, fmt.Errorf("invalid User/login invalid: %v", err)
	}
	rows, err := db.Query("SELECT BookID, Quantity FROM InShoppingCart WHERE UserID = ?", user.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("getShoppingChartBooks1: %v", err)
	}
	var books []Book
	var counts []int32
	for rows.Next() {
		var bookID int32
		var count int32
		err := rows.Scan(&bookID, &count)
		if err != nil {
			return nil, nil, fmt.Errorf("getShoppingChartBooks2: %v", err)
		}
		book, err := GetBookById(bookID)

		fmt.Println("book databsefunc", book)
		if err != nil {
			return nil, nil, fmt.Errorf("getShoppingChartBooks3: %v", err)
		}
		books = append(books, book)
		fmt.Println(books, "book: ", book)
		counts = append(counts, count)
	}
	return books, counts, nil
}

/*
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
		fmt.Println("|", b.Title, "|", edition, "|", b.StockAmount, "|")

	}
}
*/

func GetBookById(bookID int32) (Book, error) {
	rows, err := db.Query("SELECT Id, Title, SellerID, Edition, Description, StockAmount, Available, ISBN, NumRatings, SumRatings, Price FROM Books WHERE Id = ?", bookID)
	if err != nil {
		return Book{}, fmt.Errorf("getBookById1: %v", err)
	}
	var book Book
	for rows.Next() {
		err := rows.Scan(&book.BookID, &book.Title, &book.SellerID, &book.Edition, &book.Description, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings, &book.Price)
		if err != nil {
			return Book{}, fmt.Errorf("getBookById2: %v", err)
		}
	}
	return book, nil
}
