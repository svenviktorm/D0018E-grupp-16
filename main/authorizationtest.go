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

	var auth Authorization

	auth, err = authorizationCheck(1, "wrongpassword")
	fmt.Println("Result of testing with wrong password:")
	fmt.Println("auth: ", auth, "error:", err)
	auth, err = authorizationCheck(1, "sellerPwd")
	fmt.Println("Result of testing with right password, seller:")
	fmt.Println("auth: ", auth, "error:", err)
	auth, err = authorizationCheck(4, "")
	fmt.Println("Result of testing with nonexistent user ID:")
	fmt.Println("auth: ", auth, "error:", err)
	auth, err = authorizationCheck(2, "buyerPwd")
	fmt.Println("Result of testing with right password, buyer:")
	fmt.Println("auth: ", auth, "error:", err)
	auth, err = authorizationCheck(16, "adminPwd")
	fmt.Println("Result of testing with right password, admin:")
	fmt.Println("auth: ", auth, "error:", err)
	auth, err = authorizationCheck(17, "adminPwd2")
	fmt.Println("Result of testing with right password, future but not yet admin:")
	fmt.Println("auth: ", auth, "error:", err)

	fmt.Println("Trying to promote to admin:")
	fmt.Println("Trying with a non-admin user:")
	err = promoteToAdmin(17, 1, "sellerPwd")
	fmt.Println("err:", err)
	auth, err = authorizationCheck(17, "adminPwd2")
	fmt.Println("auth: ", auth, "error:", err)
	fmt.Println("Trying with a non-existent user:")
	err = promoteToAdmin(17, 4, "")
	fmt.Println("err:", err)
	auth, err = authorizationCheck(17, "adminPwd2")
	fmt.Println("auth: ", auth, "error:", err)
	fmt.Println("Trying with an admin user but wrong password:")
	err = promoteToAdmin(17, 16, "sellerPwd")
	fmt.Println("err:", err)
	auth, err = authorizationCheck(17, "adminPwd2")
	fmt.Println("auth: ", auth, "error:", err)
	fmt.Println("Trying with an admin user and correct password:")
	err = promoteToAdmin(17, 16, "adminPwd")
	fmt.Println("err:", err)
	auth, err = authorizationCheck(17, "adminPwd2")
	fmt.Println("auth: ", auth, "error:", err)

}
