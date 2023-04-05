// app.go

package main

import (
	"database/sql"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	_ "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	Router    *mux.Router
	DB        *sql.DB
	JwtSecret string
}

func (a *App) Initialize(user, password, dbname, jwtSecret string) {
	connectionString :=
		fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.JwtSecret = jwtSecret
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(":8010", a.Router))
}

func (a *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	p := product{ID: id}
	if err := p.getProduct(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Product not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, p)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *App) getProducts(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.FormValue("count"))
	start, _ := strconv.Atoi(r.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}
	if start < 0 {
		start = 0
	}

	products, err := getProducts(a.DB, start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, products)
}

func (a *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if err := p.createProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, p)
}

func (a *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
		return
	}
	defer r.Body.Close()
	p.ID = id

	if err := p.updateProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, p)
}

func (a *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	p := product{ID: id}
	if err := p.deleteProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

// Healthcheck endpoint that complies with Java Microprofile Health specification
func (a *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	dbErr := a.DB.Ping()
	if dbErr != nil {
		respondWithError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	health := map[string]interface{}{
		"status": "UP",
		"checks": []map[string]interface{}{
			{
				"name":   "database",
				"status": "UP",
			},
		},
	}

	respondWithJSON(w, http.StatusOK, health)
}

// Generate a JWT token
func (a *App) generateJWT(username, role string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute) // token expires in 5 minutes

	claims := jwt.MapClaims{
		"exp":      expirationTime.Unix(),
		"iss":      "go-mux",
		"sub":      role,
		"username": username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(a.JwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Middleware to check if the request has a valid JWT token
func (a *App) jwtAuthentication(requiredRoles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			respondWithError(w, http.StatusUnauthorized, "No token provided")
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.JwtSecret), nil
		})

		if err != nil || !token.Valid {
			respondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		if !contains(requiredRoles, claims["sub"].(string)) {
			respondWithError(w, http.StatusUnauthorized, "Insufficient permissions")
			return
		}

		next(w, r)
	}
}

// helper function to check if a string is in a slice
func contains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func (a *App) generateToken(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Replace this with proper authentication logic
	// hardcoded users => could be replaced with users from a database
	if creds.Username == "admin" && creds.Password == "admin_password" {
		token, err := a.generateJWT("admin", "admin")
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]string{"token": token})
	} else if creds.Username == "moderator" && creds.Password == "moderator_password" {
		token, err := a.generateJWT("moderator", "moderator")
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]string{"token": token})
	} else {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
	}
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/products", a.getProducts).Methods("GET")
	a.Router.HandleFunc("/product/{id:[0-9]+}", a.getProduct).Methods("GET")
	// use jwtAuthentication middleware to protect the following endpoints
	a.Router.HandleFunc("/product", a.jwtAuthentication([]string{"admin", "moderator"}, a.createProduct)).Methods("POST")
	a.Router.HandleFunc("/product/{id:[0-9]+}", a.jwtAuthentication([]string{"admin", "moderator"}, a.updateProduct)).Methods("PUT")
	a.Router.HandleFunc("/product/{id:[0-9]+}", a.jwtAuthentication([]string{"admin"}, a.deleteProduct)).Methods("DELETE")
	// additional endpoints
	a.Router.HandleFunc("/health", a.healthCheck).Methods("GET")
	a.Router.HandleFunc("/token", a.generateToken).Methods("POST")
}
