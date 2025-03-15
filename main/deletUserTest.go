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

	/*
			username := "seller10"
			password := "sellerPwd10"

			otherNonAdminUsername := "aBuyer"
			oNAPassword := "buyerPwd"
			user, success, err := LogInCheckNotHashed(username, password)
			userID=user.UserID
			fmt.Println(user, success, err)
			otherNAuser, success, err := LogInCheckNotHashed(otherNonAdminUsername, oNAPassword)


				fmt.Println("Trying to delete user", user.UserID, " with an unathourized account")
				err = deleteUserAccount(user.UserID, otherNAuser.UserID, oNAPassword)
				user, success, err = LogInCheckNotHashed(username, password)
				fmt.Println(err)
				fmt.Println(user, success, err)
				books, err := GetSellerBooks(user.UserID)
				fmt.Println(books)
				fmt.Println(err)

				fmt.Println("Trying to delete user", user.UserID, " from the same account with the wrong password")
				err = deleteUserAccount(user.UserID, user.UserID, oNAPassword)
				fmt.Println(err)
				user, success, err = LogInCheckNotHashed(username, password)
				fmt.Println(user, success, err)
				books, err = GetSellerBooks(user.UserID)
				fmt.Println(books)
				fmt.Println(err)

				fmt.Println("Trying to delete user", user.UserID, " from the same account with the right password")
				err = deleteUserAccount(user.UserID, user.UserID, password)
				fmt.Println(err)
				user, success, err = LogInCheckNotHashed(username, password)
				fmt.Println(user, success, err)

		books, err := GetSellerBooks(userID)//OBS! cant use user.UserID since if everything went right tthe logincheck failed
		fmt.Println(books)
		fmt.Println(err)

	*/

	removeUserID := int32(15)
	adminUsername := "anAdmin"
	adminPwd := "adminPwd"
	wrongPwd := "blbgw"

	adminuser, success, err := LogInCheckNotHashed(adminUsername, adminPwd)
	fmt.Println(adminuser, success, err)
	removeUser, err := GetUserByID(removeUserID)
	fmt.Println(removeUser)

	fmt.Println("Trying to delete user", removeUserID, " from and admin account with the wrong password")
	err = deleteUserAccount(removeUserID, adminuser.UserID, wrongPwd)
	fmt.Println(err)
	user, err := GetUserByID(removeUserID)
	fmt.Println(user, err)

	fmt.Println("Trying to delete user", removeUserID, " from and admin account with the right password")
	err = deleteUserAccount(removeUserID, adminuser.UserID, adminPwd)
	fmt.Println(err)
	user, err = GetUserByID(removeUserID)
	fmt.Println(user, err)

}
