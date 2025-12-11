package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/kn1ghtm0nster/handlers"
	"github.com/kn1ghtm0nster/internal/auth"
	"github.com/kn1ghtm0nster/internal/database"
	"github.com/kn1ghtm0nster/utils"
)

type User struct {
	ID 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Email 		string 		`json:"email"`
	IsChirpyRed bool		`json:"is_chirpy_red"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type CreateChirpRequest struct {
	Body   string    `json:"body"`
}

type Chirp struct {
	ID 	  		uuid.UUID `json:"id"`
	CreatedAt 	time.Time `json:"created_at"`
	UpdatedAt 	time.Time `json:"updated_at"`
	Body      	string    `json:"body"`
	UserID    	uuid.UUID `json:"user_id"`
}

type LoginRequest struct {
	Email    		 string 	`json:"email"`
	Password 		 string 	`json:"password"`
}

type LoginResponse struct {
	ID 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Email 		string 		`json:"email"`
	Token		string		`json:"token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	IsChirpyRed bool		`json:"is_chirpy_red"`
}

type WebHookData struct {
	UserID 	uuid.UUID `json:"user_id"`
}

type WebHook struct {
	Event	string `json:"event"`
	Data WebHookData `json:"data"`
}

type apiConfig struct {
	fileserverHits 	atomic.Int32
	db 				*database.Queries
	platform 		string
	secret 			string
	polkaKey		string
}

func (cfg *apiConfig) polkaWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var webhookReq WebHook

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || apiKey != cfg.polkaKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&webhookReq)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if webhookReq.Event != "user.upgraded" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.db.UpgradeUserChirpyRed(r.Context(), webhookReq.Data.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
	
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	currCount := cfg.fileserverHits.Load()
	payload := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, currCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(payload))
}

func (cfg *apiConfig) resetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// check platform
	if cfg.platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// reset users table
	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {

	// Parse request body
	var req CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// ensure password is not empty
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ensure email is not empty
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// create the new user
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email: 			req.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// map returned user to response struct
	resp := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	// send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateChirpRequest

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.Body == "" {
		http.Error(w, "Body is required", http.StatusBadRequest)
		return
	}

	// clean the body
	cleanedBody := utils.CleanProfanity(req.Body)

	// ensure length is less than 140 chars
	if len(cleanedBody) > 140 {
		http.Error(w, "Chirp is too long", http.StatusBadRequest)
		return
	}

	newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := Chirp{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserID:    newChirp.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	authorID := r.URL.Query().Get("author_id")

	var chirps []database.Chirp
	var err error

	if authorID != "" {
		parsedID, parseErr := uuid.Parse(authorID)
		if parseErr != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		chirps, err = cfg.db.GetChirpsByAuthorId(r.Context(), parsedID)
	} else {
		chirps, err = cfg.db.GetAllChirps(r.Context())
	}

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]Chirp, len(chirps))

	for i, chirp := range chirps {
		resp[i] = Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) getChirpByIdHandler(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")

	parsedId, err := uuid.Parse(chirpID)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), parsedId)
	// handle not found errors on top of other possible errors
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Chirp not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) deleteChirpByIdHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation for deleting a chirp by ID
	chirpID := r.PathValue("chirpID")

	parsedId, err := uuid.Parse(chirpID)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), parsedId)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Chirp not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if chirp.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteChirpById(r.Context(), database.DeleteChirpByIdParams{
		ID:     parsedId,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation for user login
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	match, err := auth.CheckPasswordHash(req.Password, user.HashedPassword)
	if err != nil || !match {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	createdRefreshToken, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		UserID: user.ID,
		Token: refreshToken,
	})

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:    	token,
		RefreshToken: createdRefreshToken.Token,
		IsChirpyRed: user.IsChirpyRed,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Extract refresh token from Authorization header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Look up user from refresh token provided
	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Generate new access token for that user (1 hour expiry)
	newAccessToken, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 4. Return new access token in response
	resp := map[string]string{
		"token": newAccessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (cfg *apiConfig) revokeRefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Extract refresh token from Authorization header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Revoke the refresh token in the database
	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) updateUserEmailPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserRequest

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	newHashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	updatedUser, err := cfg.db.UpdateUserEmailPassword(r.Context(), database.UpdateUserEmailPasswordParams{
		ID: userID,
		Email: req.Email,
		HashedPassword: newHashedPassword,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := User{
		ID:        updatedUser.ID,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
		Email:     updatedUser.Email,
		IsChirpyRed: updatedUser.IsChirpyRed,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}


func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	dbQueries := database.New(db)
	platform := os.Getenv("PLATFORM")
	secret := os.Getenv("SECRET")
	polkaKey := os.Getenv("POLKA_KEY")
	port := 8080
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr: fmt.Sprintf(":%d", port),
	}
	apiConfig := &apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: platform,
		secret: secret,
		polkaKey: polkaKey,
	}

	mux.HandleFunc("POST /api/users", apiConfig.createUserHandler)
	mux.HandleFunc("PUT /api/users", apiConfig.updateUserEmailPasswordHandler)
	mux.HandleFunc("POST /api/login", apiConfig.loginHandler)
	mux.HandleFunc("POST /api/refresh", apiConfig.refreshTokenHandler)
	mux.HandleFunc("POST /api/revoke", apiConfig.revokeRefreshTokenHandler)
	mux.HandleFunc("POST /api/chirps", apiConfig.createChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiConfig.getAllChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiConfig.getChirpByIdHandler)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiConfig.deleteChirpByIdHandler)
	mux.HandleFunc("POST /api/polka/webhooks", apiConfig.polkaWebhookHandler)
	mux.HandleFunc("GET /api/healthz", handlers.ReadinessHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.resetMetricsHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiConfig.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
    mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Println("Listening on port:", port)
	log.Fatal(server.ListenAndServe())
}