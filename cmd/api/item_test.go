package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/mailer"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   int32     `json:"runtime,omitempty,string"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

type MovieModel struct {
	DB *sql.DB
}

var cfg Config

func Start() {
	flag.IntVar(&cfg.port, "port", 9000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// in powershell use next command: $env:DSN="postgres://postgres:postgres@localhost:5432/greenlight?sslmode=disable"
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:postgres@localhost:5432/aitu_golang?sslmode=disable", "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")

	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "0abf276416b183", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "d8672aa2264bb5", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")
	flag.Parse()

}

func TestMovieModelInsert(t *testing.T) {
	Start()
	if cfg.port != 9000 {
		t.Errorf("The port is wrong, expected %d got %d", 9000, cfg.port)
	}

	// flag.Parse()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}
	movieModel := &MovieModel{DB: db}

	expectedMovie := &Movie{
		Title:   "The Matrix",
		Year:    1999,
		Runtime: 136,
		Genres:  []string{"Action", "Sci-Fi"},
	}

	// deleteQuery := fmt.Sprintf("delete from movies where title='%s'", expectedMovie.Title)
	// _, err = movieModel.DB.Exec(deleteQuery)

	err = app.models.Movies.Insert((*data.Movie)(expectedMovie))
	if err != nil {
		t.Errorf("Error. expected: %s", expectedMovie.Title)
	}

	query := fmt.Sprintf("select from movies where title='%s'", expectedMovie.Title)
	_, err = movieModel.DB.Exec(query)
	if err != nil {
		t.Errorf("Error. There is no this movie in database with name %s", expectedMovie.Title)
	}
}

func TestGetMovieByTitle(t *testing.T) {

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}
	movieModel := &MovieModel{DB: db}

	expectedMovie := &Movie{
		ID:      15,
		Title:   "Test movie",
		Year:    1999,
		Runtime: 136,
		Genres:  []string{"Action", "Sci-Fi"},
	}

	deleteQuery := fmt.Sprintf("delete from movies where title='%s'", expectedMovie.Title)
	_, err = movieModel.DB.Exec(deleteQuery)

	err = app.models.Movies.Insert((*data.Movie)(expectedMovie))
	if err != nil {
		fmt.Printf("Error. expected: %s", expectedMovie.Title)
	}

	movie, movieErr := app.models.Movies.GetByTitle(expectedMovie.Title)
	if movieErr != nil {
		fmt.Println("Database error")
	}

	if movie.Title != expectedMovie.Title {
		fmt.Printf("Error. Expected movie: %s, movie which we get: %s\n", expectedMovie.Title, movie.Title)
	}

	err = app.models.Movies.DeleteByTitle(expectedMovie.Title)
}

func TestDeleteMovieByTitle(t *testing.T) {

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")

	movieModel := &MovieModel{DB: db}
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}
	expectedMovie := &Movie{
		ID:      15,
		Title:   "Test movie",
		Year:    1999,
		Runtime: 136,
		Genres:  []string{"Action", "Sci-Fi"},
	}

	deleteQuery := fmt.Sprintf("delete from movies where title='%s'", expectedMovie.Title)
	_, err = movieModel.DB.Exec(deleteQuery)

	err = app.models.Movies.Insert((*data.Movie)(expectedMovie))
	if err != nil {
		fmt.Printf("Error. expected: %s", expectedMovie.Title)
	}

	err = app.models.Movies.DeleteByTitle(expectedMovie.Title)

	_, movieErr := app.models.Movies.GetByTitle(expectedMovie.Title)
	if movieErr == nil {
		fmt.Println("Movie wasn't delete")
	}
}

func (app *application) TestGetAllMovies(t *testing.T) {
	var cfg Config
	flag.IntVar(&cfg.port, "port", 8000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DSN"), "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")

	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "0abf276416b183", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "d8672aa2264bb5", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Parse()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")

	// movieModel := &MovieModel{DB: db}

	movies, err := app.models.Movies.GetAllMovies()
	if err != nil {
		fmt.Println("Database error")
	}

	if len(movies) != 5 {
		fmt.Printf("Error; Expected 5 movies, but was %d", len(movies))
	}
}

// func (app *application) TestUpdateMovie(t *testing.T) {
// 	var cfg Config
// 	flag.IntVar(&cfg.port, "port", 8000, "API server port")
// 	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

// 	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DSN"), "PostgreSQL DSN")

// 	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
// 	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
// 	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")

// 	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
// 	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
// 	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

// 	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
// 	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
// 	flag.StringVar(&cfg.smtp.username, "smtp-username", "0abf276416b183", "SMTP username")
// 	flag.StringVar(&cfg.smtp.password, "smtp-password", "d8672aa2264bb5", "SMTP password")
// 	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

// 	flag.Parse()
// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")

// 	movieModel := &MovieModel{DB: db}

// 	testMovie := &Movie{
// 		ID:      15,
// 		Title:   "TheMatrix",
// 		Year:    1999,
// 		Runtime: 136,
// 		Genres:  []string{"Action", "Sci-Fi"},
// 	}

// 	updatedMovie := &Movie{
// 		ID:      15,
// 		Title:   "TestMovie",
// 		Year:    2003,
// 		Runtime: 97,
// 		Genres:  []string{"Action", "Sci-Fi"},
// 	}

// 	deleteQuery := fmt.Sprintf("delete from movies where title='%s'", testMovie.Title)
// 	_, err = movieModel.DB.Exec(deleteQuery)

// 	err = app.models.Movies.Insert((*data.Movie)(testMovie))
// 	if err != nil {
// 		fmt.Printf("Error. expected: %s", testMovie.Title)
// 	}

// 	err = app.models.Movies.Update((*data.Movie)(testMovie))
// 	if err != nil {
// 		fmt.Println("Movie wasn't updated")
// 		fmt.Println(err)
// 	}

// 	movie, movieErr := app.models.Movies.GetByTitle("TestMovie")
// 	if movieErr != nil || movie.Version != 2 {
// 		fmt.Println("Database error")
// 	}

// 	if movie.Title != updatedMovie.Title {
// 		fmt.Printf("Error; Expected %s but was %s", updatedMovie.Title, movie.Title)
// 	}

// 	_ = app.models.Movies.DeleteByTitle(updatedMovie.Title)
// }

func TestOpenDB(t *testing.T) {
	// logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Errorf("OpenDB returned error: %v", err)
	}
	defer db.Close()

	if db.Stats().MaxOpenConnections != cfg.db.maxIdleConns {
		t.Errorf("Expected max idle connections to be %d, but got %d", cfg.db.maxIdleConns, db.Stats().MaxOpenConnections)
	}

	if db.Stats().MaxOpenConnections != cfg.db.maxOpenConns {
		t.Errorf("Expected max open connections to be %d, but got %d", cfg.db.maxOpenConns, db.Stats().MaxOpenConnections)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Errorf("Ping returned error: %v", err)
	}
}

func TestServerErrorResponse(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	app.serverErrorResponse(rr, req, err)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected status code %d but got %d", http.StatusInternalServerError, status)
	}

	expectedBody := "{\"error\":\"the server encountered a problem and could not process your request\"}\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected response body %q but got %q", expectedBody, body)
	}
}

func (app *application) TestNotFoundResponse(t *testing.T) {
	var cfg Config
	flag.IntVar(&cfg.port, "port", 8000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DSN"), "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")

	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "0abf276416b183", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "d8672aa2264bb5", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}

	defer db.Close()
	logger.Printf("database connection pool established")

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	app.notFoundResponse(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %d but got %d", http.StatusNotFound, status)
	}

	expectedBody := "{\"error\":\"the requested resource could not be found\"}\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected response body %q but got %q", expectedBody, body)
	}
}
