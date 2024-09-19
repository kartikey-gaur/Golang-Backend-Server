package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

// User model
type User struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Post model
type Post struct {
	PostID   int       `json:"postId"`
	Caption  string    `json:"caption"`
	ImageURL string    `json:"image_url"`
	PostedAt time.Time `json:"posted_at"`
	UserID   string    `json:"userId"`
}

// Initialize database connection
func startDB() {
	var err error
	connStr := "user=postgres password=kaku@4991 dbname=codesmithdb sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Unable to connect to the database:", err)
	}
	fmt.Println("Database connected!")
}

// Create a user
func PasswordEncryptor(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// createUser handles the creation of a new user with hashed password.
func creatingUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := PasswordEncryptor(user.Password)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Update user.Password with the hashed password
	user.Password = hashedPassword

	// Insert user into the database
	query := `INSERT INTO users (userId, name, email, password) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(query, user.UserID, user.Name, user.Email, user.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the created user
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Get a user by ID
func getTheUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["userId"]

	var user User
	query := `SELECT userId, name, email FROM users WHERE userId=$1`
	err := db.QueryRow(query, userID).Scan(&user.UserID, &user.Name, &user.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// Create a post
func creatingPost(w http.ResponseWriter, r *http.Request) {
	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	post.PostedAt = time.Now()

	query := `INSERT INTO posts (caption, image_url, posted_at, userId) VALUES ($1, $2, $3, $4) RETURNING postId`
	err = db.QueryRow(query, post.Caption, post.ImageURL, post.PostedAt, post.UserID).Scan(&post.PostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

// Get a post by ID
func getThePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	postID := params["postId"]

	var post Post
	query := `SELECT postId, caption, image_url, posted_at, userId FROM posts WHERE postId=$1`
	err := db.QueryRow(query, postID).Scan(&post.PostID, &post.Caption, &post.ImageURL, &post.PostedAt, &post.UserID)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(post)
}

// List of all posts by user
func getAllPostsOfUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["userId"]

	rows, err := db.Query(`SELECT postId, caption, image_url, posted_at FROM posts WHERE userId=$1`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.PostID, &post.Caption, &post.ImageURL, &post.PostedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	json.NewEncoder(w).Encode(posts)
}

func main() {
	// Initialize the database
	startDB()
	defer db.Close()

	// Create a new router
	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/users", creatingUser).Methods("POST")
	r.HandleFunc("/users/{userId}", getTheUser).Methods("GET")
	r.HandleFunc("/posts", creatingPost).Methods("POST")
	r.HandleFunc("/posts/{postId}", getThePost).Methods("GET")
	r.HandleFunc("/posts/users/{userId}", getAllPostsOfUser).Methods("GET")

	// Server initiation
	log.Fatal(http.ListenAndServe(":8888", r))
}
