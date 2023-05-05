package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/mailer"
	"github.com/stretchr/testify/require"
)

func makeRequest(method, url string, requestBody interface{}, headers http.Header) (*httptest.ResponseRecorder, error) {
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

	var reqBody []byte
	if requestBody != nil {
		var err error
		reqBody, err = json.Marshal(requestBody)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header = headers

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.listMoviesHandler)
	handler.ServeHTTP(rr, req)

	return rr, nil
}

func TestCreateMovieHandler(t *testing.T) {
	Start()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	server := httptest.NewServer(app.routes())
	defer server.Close()

	movie := &data.Movie{
		Title:   "Movie 1",
		Year:    1999,
		Runtime: 64,
		Genres:  []string{"Drama", "Sci-Fi"},
	}

	jsonMovie, err := json.Marshal(movie)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/movies", bytes.NewBuffer(jsonMovie))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var envelope struct {
		Movie *data.Movie `json:"movie"`
	}

	err = json.NewDecoder(res.Body).Decode(&envelope)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteMovieHandler(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	// Create a movie to be deleted
	movie := &data.Movie{
		Title:   "Movie 1",
		Year:    1999,
		Runtime: 64,
		Genres:  []string{"Drama", "Sci-Fi"},
	}
	err = app.models.Movies.Insert(movie)
	if err != nil {
		t.Fatal(err)
	}

	// Start a test server
	server := httptest.NewServer(app.routes())
	defer server.Close()

	// Send a DELETE request to delete the movie
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/movies/%d", server.URL, movie.ID), nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// Check that the response status code is 200 OK
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d but got %d", http.StatusOK, res.StatusCode)
	}

	// Check that the response body contains the expected message
	var envelope struct {
		Message string `json:"message"`
	}
	err = json.NewDecoder(res.Body).Decode(&envelope)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Message != "movie successfully deleted" {
		t.Errorf("expected message 'movie successfully deleted' but got '%s'", envelope.Message)
	}

	// Check that the movie was deleted from the database
	_, err = app.models.Movies.Get(movie.ID)
	if err == nil || !errors.Is(err, data.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound but got %v", err)
	}
}

func TestShowMovieHandler(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	server := httptest.NewServer(app.routes())
	defer server.Close()

	// Create a movie to retrieve later
	movie := &data.Movie{
		Title:   "Movie 1",
		Year:    1999,
		Runtime: 64,
		Genres:  []string{"Drama", "Sci-Fi"},
	}

	err = app.models.Movies.Insert(movie)
	if err != nil {
		t.Fatal(err)
	}

	// Make a request to retrieve the movie
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/movies/%d", server.URL, movie.ID), nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var envelope struct {
		Movie *data.Movie `json:"movie"`
	}

	err = json.NewDecoder(res.Body).Decode(&envelope)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the retrieved movie matches the created movie
	if envelope.Movie.ID != movie.ID {
		t.Errorf("expected movie ID %d, but got %d", movie.ID, envelope.Movie.ID)
	}

	if envelope.Movie.Title != movie.Title {
		t.Errorf("expected movie title %q, but got %q", movie.Title, envelope.Movie.Title)
	}

	if envelope.Movie.Year != movie.Year {
		t.Errorf("expected movie year %d, but got %d", movie.Year, envelope.Movie.Year)
	}

	if !reflect.DeepEqual(envelope.Movie.Genres, movie.Genres) {
		t.Errorf("expected movie genres %v, but got %v", movie.Genres, envelope.Movie.Genres)
	}
}

func TestServe(t *testing.T) {
	// cfg := getConfig()
	// app := newApplication(cfg)
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	server := httptest.NewServer(app.routes())
	defer server.Close()

	// Create a GET request to the health check endpoint
	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %d; got %d", http.StatusOK, res.StatusCode)
	}
}

func TestMethodNotAllowedResponse(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &application{
		logger: logger,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.methodNotAllowedResponse(w, r)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d; got %d", http.StatusMethodNotAllowed, res.StatusCode)
	}

	expectedMessage := "{\"error\":\"the GET method is not supported for this resource\"}\n"
	actualMessage, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualMessage) != expectedMessage {
		t.Errorf("expected message %q; got %q", expectedMessage, string(actualMessage))
	}
}

func TestApplication_editConflictResponse(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	server := httptest.NewServer(app.routes())
	defer server.Close()

	movie := &data.Movie{
		Title:   "Test Movie",
		Year:    2022,
		Runtime: 90,
		Genres:  []string{"Action", "Adventure"},
	}
	err = app.models.Movies.Insert(movie)
	require.NoError(t, err)

	// Simulate another user updating the movie
	movie.Version++
	err = app.models.Movies.Update(movie)
	require.NoError(t, err)

	// Attempt to update the movie again and expect an edit conflict response
	reqBody := `{"title": "New Title", "year": 2023, "runtime": 100, "genres": ["Action", "Drama"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/movies/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// app.srv.Handler.ServeHTTP(rr, req)
	server.Client().Do(req)

	// Expect a 409 Conflict response with the expected error message
	expectedResponse := envelope{"error": "unable to update the record due to an edit conflict, please try again"}
	assertResponse(t, rr, http.StatusConflict, expectedResponse)
}

func assertResponse(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantBody interface{}) {
	t.Helper()

	if rr.Code != wantStatus {
		t.Errorf("want status %d; got %d", wantStatus, rr.Code)
	}

	if wantBody != nil {
		var gotBody map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&gotBody)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(gotBody, wantBody) {
			t.Errorf("want body %#v; got %#v", wantBody, gotBody)
		}
	}
}
