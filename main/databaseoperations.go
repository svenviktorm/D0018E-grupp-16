package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"golang.org/x/crypto/sha3"
)

// ---------------STRUCT TYPES---------------------------
type User struct {
	UserID   int32
	Username sql.NullString
	Password string
	Email    sql.NullString
	IsAdmin  bool
	IsSeller bool
	IsActive bool
}

type Authorization struct {
	//This is a 'smaller version of User' for when we only want to know if the user have the right to do something or not
	UserID          int32
	AuthorizationOK bool //the user credentials have been (sucessfully) checked
	IsAdmin         bool
	IsSeller        bool
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

	Author      string         `json:"author"`
	Image       sql.NullString `json:"image"`
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

type Order struct {
	OrderID             int32
	SellerID            int32
	CustomerID          int32
	TimeEntered         []uint8
	TimeConfirmed       sql.NullTime
	TimeSent            sql.NullTime
	TimePaymentReceived sql.NullTime
	PaymentReceived     bool
	PaymentMethod       string
	Status              string
	DeliveryAddress     sql.NullString
	BillingAddress      sql.NullString
}

// enum for orderStatus
const (
	OrderStatusReserved  = "reserved"
	OrderStatusConfirmed = "confirmed"
	//OrderStatusPayed     = "payed"
	OrderStatusSent     = "sent"
	OrderStatusCanceled = "canceled"
	OrderStatusReturned = "returned"
)

// enum for paymentMethod
const (
	paymentMethodInvoice = "invoice"
	paymentMethodCard    = "card"
)

const returnsAllowedTime = 14 //How long after delivery a purchase can be refunded

type errorType int

const (
	errorTypeAuthorizationNotFound     errorType = -1
	errorTypeAuthorizationUnauthorized errorType = -2
	errorTypeUserNotFound              errorType = -3
	errorTypeDatabase                  errorType = -4
	errorTypeBadRequest                errorType = -5
	errorTypeConflict                  errorType = -6
)

type MyError struct {
	inFunction string
	errorText  string
	errorType  errorType
}

func (e MyError) Error() string {
	return e.inFunction + ":" + e.errorText
}

//---------------USER ACCOUNTS---------------------------

func hash(plaintext string) int64 {
	h := sha3.New256()
	h.Write([]byte(plaintext))
	hash := h.Sum(nil)
	//first 64 bits
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func AddUser(username string, password string, email sql.NullString) (int32, error) {
	fmt.Println("kalle")
	var passwordHash int64 = hash(password)
	result, err := db.Exec("INSERT INTO Users (username, PasswordHash, email) VALUES (?, ?, ?)", username, passwordHash, email)
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
	/*
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
	*/
	fmt.Println("anka")
	return i32, nil
}

// for checking if a username and password exists
func LogInCheckNotHashed(username string, password string) (user User, loginSucces bool, err error) {
	var passwordHash int64 = hash(password)
	return LoginCheck(username, passwordHash)
}

func authorizationCheck(userID int32, password string) (Authorization, error) {
	//This does the same as LogInCheckNotHashed but with usedID instead of username, which should be faster since that's the primary key of the table
	//(for login we need to use the username but after that when checking that the user is authorized to do something we might as well use the userID.
	//this will be used a lot for the admin functions.)
	//It also returns less data about the user, only that which is relevant for wheter the user have the right to do something.
	var passwordHash int64 = hash(password)
	var authorization Authorization = Authorization{UserID: userID}
	rows, err := db.Query("SELECT IsAdmin, IsSeller FROM Users WHERE Id = ? AND PasswordHash = ? AND IsActive", userID, passwordHash)
	if err != nil {

		return authorization, fmt.Errorf("error in authorizationCheck, query error:  %v", err)
	}
	defer rows.Close()
	canReadRow := rows.Next()
	//fmt.Println("canreadrow:", canReadRow)
	if canReadRow {
		err := rows.Scan(&authorization.IsAdmin, &authorization.IsSeller)
		if err != nil {
			err = fmt.Errorf("error in authorizationCheck, couldn't scan:  %v", err)
		} else {
			authorization.AuthorizationOK = true
		}
	} else {
		//either user wasn't found (either userID doesn't exist or password incorrect) or
		//fmt.Println("rows.Err:", rows.Err())
		if rows.Err() != nil { //something went wrong when preparing to read first row
			err = fmt.Errorf("error in authorizationCheck, something wrong when preparing rows:  %v", rows.Err())
		}

	}
	//There where no such user found
	return authorization, err
}

// for checking a username with an already hashed password
func LoginCheck(username string, passwordHash int64) (user User, loginSuccess bool, err error) {
	loginSuccess = false
	user.Username = sql.NullString{String: username, Valid: true}
	err = nil
	row := db.QueryRow("SELECT Id, Email, IsAdmin, IsSeller, IsActive FROM Users WHERE Username = ? AND PasswordHash = ? AND IsActive", username, passwordHash)

	err = row.Scan(&user.UserID, &user.Email, &user.IsAdmin, &user.IsSeller, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows { // this is not an error, just a failed login (user does not exist or password is incorrect)
			user = User{}
			err = nil
		} else {
			err = fmt.Errorf("LoginCheck: database error: %v", err)
			user = User{}
		}
	} else {
		loginSuccess = true
	}
	return user, loginSuccess, err
}

// return a user struct from their userID
func GetUserByID(userID int32) (User, error) {
	row := db.QueryRow("SELECT Id, Username, PasswordHash, Email, IsAdmin, IsSeller, IsActive FROM Users WHERE Id = ? ", userID)

	var user User
	if err := row.Scan(&user.UserID, &user.Username, &user.Password, &user.Email, &user.IsAdmin, &user.IsSeller, &user.IsActive); err != nil {
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("GetUserByID: user not found")
		} else {
			return User{}, fmt.Errorf("GetUserByID %q: %v", userID, err)
		}
	}
	// the returend password will be hashed and therefore usless and therefore removed to not cause confusion
	user.Password = ""
	//there should only be one user per ID otherwise something is wrong with the database
	return user, nil
}

func changeEmail(email sql.NullString, id int32) (sql.Result, error) {
	result, err := db.Exec("UPDATE Users SET Email = ? WHERE Id = ?", email, id)
	if err != nil {
		fmt.Println("error updating email")
		return result, err
	}
	return result, nil
}

/*
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
*/

func UpgradeToSeller(toBeSellerID int32, authorizingUserID int32, authorizingPwd string, sellerName string, description sql.NullString) (int32, error) {

	authorization, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return -1, fmt.Errorf("AddSeller: authorization error: %v", err)
	}

	if !authorization.AuthorizationOK {
		return -1, fmt.Errorf("AddSeller: authorization failed: unknown user or wrong password")
	}
	if !(authorization.UserID == toBeSellerID || authorization.IsAdmin) {
		return -2, fmt.Errorf("AddSeller: authorization failed: authorizing user do not have the right to turn this account into a seller account (can only be done by account itself or administrator)")
	}
	//Authorization is ok

	tx, dberr := db.Begin()
	//defer db.Close()
	if dberr != nil {
		return -3, fmt.Errorf("AddSeller: transaction error:", dberr)
	}

	/*
		  	var descriptionValue interface{}
			if description.Valid {
				descriptionValue = description.String
			} else {
				descriptionValue = nil
			}
	*/

	result, err := tx.Exec("INSERT INTO Sellers (Name, Id, Description) VALUES (?, ?, ?)", sellerName, toBeSellerID, description)

	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -4, fmt.Errorf("AddSeller: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -5, fmt.Errorf("AddSeller: %v", err)
	}
	tx.Exec("UPDATE Users SET IsSeller = True WHERE ID = ?", toBeSellerID)
	if err != nil {
		tx.Rollback()
		fmt.Println("rollback!!!!!!")
		return -6, fmt.Errorf("AddSeller: %v", err)
	}
	err = tx.Commit()
	if err != nil {
		return -7, fmt.Errorf("Error committing transaction:", err)
	}
	return int32(id), nil
}

func UpdateSellerInfo(sellerID int32, authorizingUserID int32, authorizingPwd string, sellerName string, description sql.NullString) error {
	authorization, err := authorizationCheck(authorizingUserID, authorizingPwd)
	inFunction := "UpdateSellerInfo"
	if err != nil {
		return MyError{inFunction: inFunction, errorText: fmt.Sprintf("authorization error: %v", err), errorType: errorTypeAuthorizationNotFound}
	}

	if !authorization.AuthorizationOK {
		return MyError{inFunction: inFunction, errorText: "authorization failed: unknown user or wrong password", errorType: errorTypeAuthorizationNotFound}
	}
	if !(authorization.UserID == sellerID || authorization.IsAdmin) {
		return MyError{inFunction: inFunction, errorText: "authorization failed: authorizing user do not have the right to turn this account into a seller account (can only be done by account itself or administrator)", errorType: errorTypeAuthorizationUnauthorized}
	}
	//Authorization is ok
	seller, err := GetUserByID(sellerID)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: "seller not found", errorType: errorTypeUserNotFound}
	}
	if !seller.IsSeller {
		return MyError{inFunction: inFunction, errorText: "supposed seller is not seller", errorType: errorTypeBadRequest}
	}
	if !seller.IsActive {
		return MyError{inFunction: inFunction, errorText: "seller account is inactive", errorType: errorTypeUserNotFound}
	}
	//Seller account is ok
	fmt.Println("sellerName, description, sellerId: ", sellerName, description, sellerID)
	_, err = db.Exec("UPDATE Sellers SET Name = ?, Description = ? WHERE Id = ?", sellerName, description, sellerID)

	if err != nil {
		fmt.Println(err)
		return MyError{inFunction: inFunction, errorText: "error when updating database", errorType: errorTypeBadRequest}
	}
	return nil
}
func UpgradeToAdmin(idToUpgrade int32, authorizingId int32, authorizingPwd string) error {
	auth, err := authorizationCheck(authorizingId, authorizingPwd)
	if err != nil {
		return fmt.Errorf("Error in promoteToAdmin: Something went wrong in authorizationCheck:%v", err)
	}
	if !auth.AuthorizationOK {
		return fmt.Errorf("promoteToAdmin failed: authorizing user not found or incorrect password")
	}
	if !auth.IsAdmin {
		return fmt.Errorf("promoteToAdmin failed: authorizing user is not an administrator")
	}
	_, err = db.Exec("UPDATE Users SET IsAdmin = ? WHERE Id = ?", true, idToUpgrade)
	if err != nil {
		return fmt.Errorf("Error in promoteToAdmin: Something went wrong when updating database:%v", err)
	}
	return nil
}

func deleteUserAccount(userIDforRemoval int32, authorizingUserID int32, authorizingPwd string) error {
	inFunction := "deleteUserAccount"
	auth, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: fmt.Sprintf("authorization error: %v", err), errorType: errorTypeAuthorizationNotFound}
	}

	if !auth.AuthorizationOK {
		return MyError{inFunction: inFunction, errorText: "authorization failed: unknown user or wrong password", errorType: errorTypeAuthorizationNotFound}
	}
	if !(auth.IsAdmin || authorizingUserID == userIDforRemoval) {
		return MyError{inFunction: inFunction, errorText: "authorization failed: authorizing user do not have the right to turn this account into a seller account (can only be done by account itself or administrator)", errorType: errorTypeAuthorizationUnauthorized}
	}
	//Deletion is authorized
	user, err := GetUserByID(userIDforRemoval)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: "could not find the account", errorType: errorTypeBadRequest}
	}
	tx, err := db.Begin()
	if err != nil {
		return MyError{inFunction: inFunction, errorText: "failed to start transaction", errorType: errorTypeDatabase}
	}
	defer tx.Rollback()
	if user.IsSeller {
		var numActiveOrders int
		row := tx.QueryRow("SELECT COUNT(Id) from ORDERS WHERE SellerID=? AND (Status IN(?,?) OR (Status=? AND  DATEDIFF(TimeSent,CURDATE())>?+2))", userIDforRemoval, OrderStatusReserved, OrderStatusConfirmed, OrderStatusSent, returnsAllowedTime)
		//TODO also check for returned and not refunded orders

		err := row.Scan(&numActiveOrders)
		if err != nil {
			if err == sql.ErrNoRows { //The query returned 0 rows. Should never happen with a count() query?
				return MyError{inFunction: inFunction, errorText: "something weird happened when checking for active orders, got 0 rows from a count query?", errorType: errorTypeDatabase}
			} else { //something else went wrong with the query or scan
				return MyError{inFunction: inFunction, errorText: fmt.Sprintf("database error when checking for active orders:  %v", row.Err()), errorType: errorTypeDatabase}
			}
		} else {
			if numActiveOrders > 0 {
				return MyError{inFunction: inFunction, errorText: "User is seller with active orders", errorType: errorTypeConflict}
			}
		}

		booklist, err := GetSellerBooks(userIDforRemoval)
		for _, book := range booklist {
			fmt.Println("removing book:", book.Title)
			_, err = tx.Exec("UPDATE Books SET Available = ? WHERE Books.Id = ?", false, book.BookID)
			if err != nil {
				return MyError{inFunction: inFunction, errorText: fmt.Sprintf("user is seller and failed to remove book with ID: %v and title: %v. Error:%v", book.BookID, book.Title, err), errorType: errorTypeConflict}
			}
		}
		fmt.Println("book removal complete")
		//Clear the seller info
		_, err = tx.Exec("UPDATE Sellers Set Name='',description='' WHERE Sellers.Id=?", userIDforRemoval)
		if err != nil {
			return MyError{inFunction: inFunction, errorText: fmt.Sprintf("something went wrong when clearing seller data: %v ", err), errorType: errorTypeDatabase}
		}
	}
	//Now we can 'delete' the account
	_, err = tx.Exec("UPDATE Users Set Username=NULL,PasswordHash=0,email=NULL,IsAdmin=false,IsActive=false WHERE Users.Id=?", userIDforRemoval)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: fmt.Sprintf("something went wrong when clearing the user data: %v ", err), errorType: errorTypeDatabase}
	}
	tx.Commit()
	return nil
}

//---------BOOKS----------

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
	result, err := db.Exec("INSERT INTO Books (Title, Author, SellerID, Edition, Description, StockAmount, Available, ISBN, NumRatings, SumRatings, Price, Image) VALUES (?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", book.Title, book.Author, user.UserID, book.Edition, book.Description, book.StockAmount, book.Available, book.ISBN, 0, 0, book.Price, book.Image)
	if err != nil {
		return -1, fmt.Errorf("addBook: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -2, fmt.Errorf("addBook: %v", err)
	}
	return int32(id), nil

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
	inFunction := "createReview"
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
	tx, err := db.Begin()
	if err != nil {
		return MyError{inFunction: inFunction, errorText: "failed to start transaction", errorType: errorTypeDatabase}
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO BookReviews (BookID, UserID, Text, Rating) VALUES (?, ?, ?, ?)", bookId, userId, text, rating)
	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}

	var numRatings int
	var sumRatings int

	/*
		err = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(Rating), 0) FROM BookReviews WHERE BookID = ?", bookId).Scan(&numRatings, &sumRatings)
		if err != nil {
			return fmt.Errorf("failed to fetch updated ratings: %v", err)
		}

		averageRating := sumRatings / numRatings
	*/

	err = tx.QueryRow("SELECT Numratings,SumRatings FROM Books WHERE Id=?", bookId).Scan(&numRatings, &sumRatings)
	if err != nil {
		return fmt.Errorf("failed to fetch old book ratings: %v", err)
	}
	numRatings = numRatings + 1
	sumRatings = sumRatings + rating
	_, err = tx.Exec("UPDATE Books SET NumRatings = ?, SumRatings = ? WHERE Id = ?", numRatings, sumRatings, bookId)
	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}
	tx.Commit()
	return nil
}

func getReviews(bookId int32) ([]BookReview, float64, error) {
	fmt.Println("getReviews called", bookId)

	var sumRatings float64
	var numRatings float64
	var avRating float64
	err := db.QueryRow("SELECT NumRatings, SumRatings FROM Books WHERE Id = ?", bookId).Scan(&numRatings, &sumRatings)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get sumRatings: %v", err)
	}
	avRating = math.Round((sumRatings/numRatings)*10) / 10

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

	return reviews, avRating, nil
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

func removeBook(available bool, bookId int32, authorizingUserID int32, authorizingPwd string) error {

	inFunction := "removeBook"
	auth, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: fmt.Sprintf("authorization error: %v", err), errorType: errorTypeAuthorizationNotFound}
	}

	if !auth.AuthorizationOK {
		return MyError{inFunction: inFunction, errorText: "authorization failed: unknown user or wrong password", errorType: errorTypeAuthorizationNotFound}
	}

	var sellerID int32
	err = db.QueryRow("SELECT SellerId FROM Books WHERE Id = ?", bookId).Scan(&sellerID)
	if err != nil {
		if err == sql.ErrNoRows { //The query returned 0 rows: book does not exist
			return fmt.Errorf("book with ID %d does not exist", bookId)
		} else {
			return fmt.Errorf("error checking sellerID and book existence: %v", err)
		}
	}
	if !(auth.IsAdmin || authorizingUserID == sellerID) {
		return MyError{inFunction: inFunction, errorText: "authorization failed: authorizing user do not have the right to remove this book (can only be done by the seller of the book or an administrator)", errorType: errorTypeAuthorizationUnauthorized}
	}

	db.Exec("UPDATE Books SET Available = ? WHERE Id = ?", available, bookId)
	return nil
}

func getSellerInfo(sellerId int32) ([]Seller, error) {
	rows, err := db.Query("SELECT Id, Name, Description FROM Sellers WHERE Id = ?", sellerId)
	if err != nil {
		return nil, fmt.Errorf("getReview1: %v", err)
	}
	defer rows.Close()
	var sellerInfos []Seller
	for rows.Next() {
		var sellerInfo Seller
		err := rows.Scan(&sellerInfo.SellerID, &sellerInfo.Name, &sellerInfo.Description)
		if err != nil {
			return nil, fmt.Errorf("getBookById2: %v", err)
		}
		sellerInfos = append(sellerInfos, sellerInfo)
		fmt.Println("sellerInfo: ", sellerInfo)
	}

	return sellerInfos, nil
}

func SearchBooksByTitle(titlesearch string, onlyAvailable bool) ([]Book, error) {

	var err error
	var rows *sql.Rows

	titlesearch = "%" + titlesearch + "%"
	if onlyAvailable {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE Available AND Title LIKE ?", titlesearch)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE Title LIKE ?", titlesearch)

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
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE Available AND MATCH(Author) AGAINST(?)", authorsearch)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE MATCH(Author) AGAINST(?)", authorsearch)

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
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE Available AND ISBN=?", isbn)
	} else {
		rows, err = db.Query("SELECT Id,Title,Author,SellerID,Edition,Description,StockAmount,Available,ISBN,NumRatings,SumRatings,Price, Image FROM Books WHERE ISBN=?", isbn)

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
		if err := rows.Scan(&b.BookID, &b.Title, &b.Author, &b.SellerID, &b.Edition, &b.Description, &b.StockAmount, &b.Available, &b.ISBN, &b.NumRatings, &b.SumRatings, &b.Price, &b.Image); err != nil {
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

	rows, err := db.Query("SELECT Id, Title, Author, Description, Price, Edition, StockAmount, Available, ISBN, NumRatings, SumRatings, SellerID, Image FROM Books WHERE SellerID = ?", sellerID)
	if err != nil {
		return nil, fmt.Errorf("getSellerBooks: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var book Book
		err := rows.Scan(&book.BookID, &book.Title, &book.Author, &book.Description, &book.Price, &book.Edition, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings, &book.SellerID, &book.Image)
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
	rows, err := db.Query("SELECT Id, Title, Author, SellerID, Edition, Description, StockAmount, Available, ISBN, NumRatings, SumRatings, Price, Image FROM Books WHERE Id = ?", bookID)
	if err != nil {
		return Book{}, fmt.Errorf("getBookById1: %v", err)
	}
	var book Book
	for rows.Next() {
		err := rows.Scan(&book.BookID, &book.Title, &book.Author, &book.SellerID, &book.Edition, &book.Description, &book.StockAmount, &book.Available, &book.ISBN, &book.NumRatings, &book.SumRatings, &book.Price, &book.Image)
		if err != nil {
			return Book{}, fmt.Errorf("getBookById2: %v", err)
		}
	}
	return book, nil
}

//------SHOPPING CART-----------------

func AddBookToShoppingCart(user User, bookID int32, count int32) (newCount int32, err error) {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
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
	var bookCount int32 = 0
	rows, err = db.Query("SELECT StockAmount FROM Books WHERE Id = ?", bookID)
	if err != nil {
		return -quantity, fmt.Errorf("AddBookToShoppingCart4: %v", err)
	}
	for rows.Next() {
		err := rows.Scan(&bookCount)
		if err != nil {
			return -quantity, fmt.Errorf("AddBookToShoppingCart5: %v", err)
		}
	}
	if bookCount < count {
		return -quantity, fmt.Errorf("AddBookToShoppingCart: Not enough books in stock")
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
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
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
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("Invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("DELETE FROM InShoppingCart WHERE UserID = ? ", user.UserID)
	return err
}

func GetShoppingChartBooks(user User) ([]Book, []int32, error) {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
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

// -------ORDERS --------------
func getOrdersBySeller(sellerID int32, authorizingUserID int32, authorizingPwd string, filter string) ([]Order, error) {
	var orders []Order
	//Check authorization
	auth, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return orders, fmt.Errorf("Error in getOrdersBySeller: Something went wrong in authorizationCheck:%v", err)
	}
	if !auth.AuthorizationOK {
		return orders, fmt.Errorf("getOrdersBySeller failed: authorizing user not found or incorrect password")
	}
	if !(auth.IsAdmin || authorizingUserID == sellerID) {
		return orders, fmt.Errorf("getOrdersBySeller failed: authorizing user do not have the right to see this sellers orders (only allowed for seller themself and administrators)")
	}
	//Authorization ok
	var rows *sql.Rows

	switch filter {
	case "all":
		rows, err = db.Query("SELECT Id, SellerID, CustomerID,TimeEntered,TimeConfirmed,TimeSent,TimePaymentReceived,PaymentReceived,PaymentMethod,Status,DeliveryAddress,BillingAddress FROM Orders WHERE SellerID = ?", sellerID)
	case OrderStatusReserved:
		rows, err = db.Query("SELECT Id, SellerID, CustomerID,TimeEntered,TimeConfirmed,TimeSent,TimePaymentReceived,PaymentReceived,PaymentMethod,Status,DeliveryAddress,BillingAddress FROM Orders WHERE SellerID = ? AND Status=?", sellerID, OrderStatusReserved)
	case OrderStatusConfirmed:
		rows, err = db.Query("SELECT Id, SellerID, CustomerID,TimeEntered,TimeConfirmed,TimeSent,TimePaymentReceived,PaymentReceived,PaymentMethod,Status,DeliveryAddress,BillingAddress FROM Orders WHERE SellerID = ? AND Status=? ", sellerID, OrderStatusConfirmed)
	case "refundable":
		rows, err = db.Query("SELECT Id, SellerID, CustomerID,TimeEntered,TimeConfirmed,TimeSent,TimePaymentReceived,PaymentReceived,PaymentMethod,Status,DeliveryAddress,BillingAddress FROM Orders WHERE SellerID = ? AND Status=? AND DATEDIFF(TimeSent,CURDATE())>?+2", sellerID, OrderStatusSent, returnsAllowedTime)
		//TODO FINISH OR REMOVE
	}
	if err != nil {
		return orders, err
	}
	defer rows.Close()
	/*	fmt.Println(rows.Columns())
		cts, err := rows.ColumnTypes()
		for i, ct := range cts {
			fmt.Println(i, ct)
		}	*/
	return extractOrdersFromRows(rows)

}

func payOrder(orderID int32, user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("UPDATE Orders SET PaymentReceived = True WHERE ID = ? AND SellerID = ?", orderID, user.UserID)
	if err != nil {
		return fmt.Errorf("payOrder: %v", err)
	}
	return nil

}

func cancelOrder(orderID int32, user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	var orderStatus string
	err = db.QueryRow("SELECT Status FROM Orders WHERE Id = ? AND (CustomerID = ? OR SellerID = ?)", orderID, user.UserID, user.UserID).Scan(&orderStatus)
	if err != nil {
		return fmt.Errorf("cancelOrder: %v", err)
	}
	if orderStatus != OrderStatusReturned && orderStatus != OrderStatusCanceled && orderStatus != OrderStatusSent {
		tx, dberr := db.Begin()
		if dberr != nil {
			return fmt.Errorf("transaction erroor:", dberr)
		}
		rows, err := db.Query("SELECT BookID, Quantity FROM Orders_books WHERE OrderID = ?", orderID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("cancelOrder1: %v", err)
		}
		for rows.Next() {
			fmt.Println("cancelOrder: Looping through books")
			var bookID int32
			var quantity int32
			err := rows.Scan(&bookID, &quantity)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("cancelOrder2: %v", err)
			}
			fmt.Println("cancelOrder: BookID: ", bookID, " Quantity: ", quantity)
			_, err = tx.Exec("UPDATE Books SET StockAmount = StockAmount + ? WHERE Id = ?", quantity, bookID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("cancelOrder3: %v", err)
			}
		}
		_, err = tx.Exec("UPDATE Orders SET Status = ? WHERE Id = ?", OrderStatusCanceled, orderID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("cancelOrder4: %v", err)
		}
		fmt.Println("cancelOrder: Order canceled")
		tx.Commit()
		return nil
	} else {
		return fmt.Errorf("cancelOrder: Order already canceled, returned or sent")
	}
}

func confirmOrder(orderID int32, user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("UPDATE Orders SET Status = ? WHERE Id = ? AND SellerID = ? AND Status = ?", OrderStatusConfirmed, orderID, user.UserID, OrderStatusReserved)
	if err != nil {
		return fmt.Errorf("confirmOrder: %v", err)
	}
	return nil
}

func sendOrder(orderID int32, user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("UPDATE Orders SET Status = ? WHERE Id = ? AND SellerID = ? AND Status = ?", OrderStatusSent, orderID, user.UserID, OrderStatusConfirmed)
	if err != nil {
		return fmt.Errorf("sendOrder: %v", err)
	}
	return nil
}

func returnOrder(orderID int32, user User) error {
	user, successLogin, err := LogInCheckNotHashed(user.Username.String, user.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	_, err = db.Exec("UPDATE Orders SET Status = ? WHERE Id = ? AND CustomerID = ? AND Status = ?", OrderStatusReturned, orderID, user.UserID, OrderStatusSent)
	if err != nil {
		return fmt.Errorf("returnOrder: %v", err)
	}
	return nil
}

func MakeShoppingCartIntoOrderReserved(userO User, billingAddress string, deliveryAddress string) error {
	user, successLogin, err := LogInCheckNotHashed(userO.Username.String, userO.Password)
	if err != nil || !successLogin {
		return fmt.Errorf("invalid User/login invalid: %v", err)
	}
	tx, dberr := db.Begin()
	if dberr != nil {
		return fmt.Errorf("transaction erroor:", dberr)
	}
	rows, err := db.Query("SELECT BookID FROM InShoppingCart WHERE UserID = ?", user.UserID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("MakeShoppingCartIntoOrder1: %v", err)
	}
	sellers := []int32{}
	for rows.Next() {
		var bookID int32
		err := rows.Scan(&bookID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("MakeShoppingCartIntoOrder2: %v", err)
		}
		rows, err := db.Query("SELECT SellerID FROM Books WHERE Id = ?", bookID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("MakeShoppingCartIntoOrder3: %v", err)
		}
		var sellerID int32
		for rows.Next() {
			err := rows.Scan(&sellerID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("MakeShoppingCartIntoOrder4: %v", err)
			}
		}
		if !slices.Contains(sellers, sellerID) {
			sellers = append(sellers, sellerID)
		}
	}

	for rows.Next() {
		var sellerID int32
		err := rows.Scan(&sellerID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("MakeShoppingCartIntoOrder2: %v", err)
		}
		sellers = append(sellers, sellerID)
	}
	for _, sellerID := range sellers {
		rows, err := db.Query("SELECT BookID, Quantity FROM InShoppingCart WHERE UserID = ?", user.UserID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("MakeShoppingCartIntoOrder3: %v", err)
		}
		if sellerID == user.UserID {
			tx.Rollback()
			fmt.Println("Error: SellerID == UserID")
			return fmt.Errorf("Cannot order from yourself")
		}
		_, err = tx.Exec("INSERT INTO Orders (SellerID, CustomerID, Status, DeliveryAddress, BillingAddress) VALUES (?, ?, ?, ?, ?)", sellerID, user.UserID, OrderStatusReserved, deliveryAddress, billingAddress)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("MakeShoppingCartIntoOrder4: %v", err)
		}
		for rows.Next() {
			var bookID int32
			var quantity int32
			err := rows.Scan(&bookID, &quantity)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("MakeShoppingCartIntoOrder4: %v", err)
			}
			prices, err := db.Query("SELECT Price, Available, SellerId FROM Books WHERE Id = ?", bookID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("MakeShoppingCartIntoOrder5: %v", err)
			}
			var price sql.NullInt32
			var available bool
			var booksellerID int32
			for prices.Next() {
				err := prices.Scan(&price, &available, &booksellerID)

				if err != nil {
					tx.Rollback()
					return fmt.Errorf("MakeShoppingCartIntoOrder6: %v", err)
				}
				if !available {
					tx.Rollback()
					return fmt.Errorf("MakeShoppingCartIntoOrderBook not available")
				}
			}
			if booksellerID == sellerID {
				_, err = tx.Exec("INSERT INTO Orders_books (OrderID, BookID, Price ,Quantity) VALUES (LAST_INSERT_ID(), ? ,?, ?)", bookID, price, quantity)
				if err != nil {

					tx.Rollback()
					return fmt.Errorf("MakeShoppingCartIntoOrder7: %v", err)
				}
				_, err = tx.Exec("UPDATE Books SET StockAmount = StockAmount - ? WHERE Id = ?", quantity, bookID)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("MakeShoppingCartIntoOrder8: %v", err)
				}
			}
		}
	}
	err = ResetShoppingCart(userO)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("MakeShoppingCartIntoOrder9: %v", err)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Error committing transaction:", err)
	}
	return nil
}
func getOrdersByBuyer(buyerID int32, authorizingUserID int32, authorizingPwd string) ([]Order, error) {
	var orders []Order
	//Check authorization
	auth, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return orders, fmt.Errorf("Error in getOrdersByBuyer: Something went wrong in authorizationCheck:%v", err)
	}
	if !auth.AuthorizationOK {
		return orders, fmt.Errorf("getOrdersByBuyer failed: authorizing user not found or incorrect password")
	}
	if !(auth.IsAdmin || authorizingUserID == buyerID) {
		return orders, fmt.Errorf("getOrdersByBuyer failed: authorizing user do not have the right to see this sellers orders (only allowed for seller themself and administrators)")
	}
	//Authorization ok
	rows, err := db.Query("SELECT Id, SellerID, CustomerID,TimeEntered,TimeConfirmed,TimeSent,TimePaymentReceived,PaymentReceived,PaymentMethod,Status,DeliveryAddress,BillingAddress FROM Orders WHERE CustomerID = ?", buyerID)
	if err != nil {
		return orders, fmt.Errorf("getOrdersByBuyer: %v", err)
	}
	defer rows.Close()

	return extractOrdersFromRows(rows)

}

func getBooksAndPriceFromOrder(orderID int32) (books []Book, prices []int32, quantities []int32, err error) {
	rows, err := db.Query("SELECT BookID, Price, Quantity FROM Orders_books WHERE OrderID = ?", orderID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getBooksAndPriceFromOrder1: %v", err)
	}
	for rows.Next() {
		var bookID int32
		var price int32
		var quantity int32
		err := rows.Scan(&bookID, &price, &quantity)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("getBooksAndPriceFromOrder2: %v", err)
		}
		book, err := GetBookById(bookID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("getBooksAndPriceFromOrder3: %v", err)
		}
		books = append(books, book)
		prices = append(prices, price)
		quantities = append(quantities, quantity)
	}
	return books, prices, quantities, nil
}

func extractOrdersFromRows(rows *sql.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.OrderID, &order.SellerID, &order.CustomerID, &order.TimeEntered,
			&order.TimeConfirmed, &order.TimeSent, &order.TimePaymentReceived, &order.PaymentReceived,
			&order.PaymentMethod, &order.Status, &order.DeliveryAddress, &order.BillingAddress)
		if err != nil {
			return orders, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

//-------REVIEWS-------------

func deleteReview(reviewId int32, authorizingUserID int32, authorizingPwd string) error {
	inFunction := "deleteReview"
	auth, err := authorizationCheck(authorizingUserID, authorizingPwd)
	if err != nil {
		return MyError{inFunction: inFunction, errorText: fmt.Sprintf("authorization error: %v", err), errorType: errorTypeAuthorizationNotFound}
	}

	if !auth.AuthorizationOK {
		return MyError{inFunction: inFunction, errorText: "authorization failed: unknown user or wrong password", errorType: errorTypeAuthorizationNotFound}
	}

	var reviewerID int32
	var bookID int32
	var rating int
	err = db.QueryRow("SELECT UserID, BookID,rating FROM BookReviews WHERE Id = ?", reviewId).Scan(&reviewerID, &bookID, &rating)
	if err != nil {
		if err == sql.ErrNoRows { //The query returned 0 rows: book does not exist
			return fmt.Errorf("review with ID %d does not exist", reviewId)
		} else {
			return fmt.Errorf("error checking reviewerID and review existence: %v", err)
		}
	}
	if !(auth.IsAdmin || authorizingUserID == reviewerID) {
		return MyError{inFunction: inFunction, errorText: "authorization failed: authorizing user do not have the right to remove this book (can only be done by the seller of the book or an administrator)", errorType: errorTypeAuthorizationUnauthorized}
	}
	//Authoriztaion ok
	tx, err := db.Begin()
	if err != nil {
		return MyError{inFunction: inFunction, errorText: "failed to start transaction", errorType: errorTypeDatabase}
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM BookReviews WHERE Id=?", reviewId)
	if err != nil {
		return fmt.Errorf("failed to delete review: %v", err)
	}

	var numRatings int
	var sumRatings int

	/*
		err = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(Rating), 0) FROM BookReviews WHERE BookID = ?", bookId).Scan(&numRatings, &sumRatings)
		if err != nil {
			return fmt.Errorf("failed to fetch updated ratings: %v", err)
		}

		averageRating := sumRatings / numRatings
	*/

	err = tx.QueryRow("SELECT Numratings,SumRatings FROM Books WHERE Id=?", bookID).Scan(&numRatings, &sumRatings)
	if err != nil {
		return fmt.Errorf("failed to fetch old book ratings: %v", err)
	}
	numRatings = numRatings - 1
	sumRatings = sumRatings - rating
	_, err = tx.Exec("UPDATE Books SET NumRatings = ?, SumRatings = ? WHERE Id = ?", numRatings, sumRatings, bookID)
	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}
	tx.Commit()
	return nil

}
