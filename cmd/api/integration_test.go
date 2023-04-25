package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/mailer"
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

// func TestRegisterUserHandlerIntegration(t *testing.T) {
// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer db.Close()

// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db),
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	server := httptest.NewServer(app.routes())
// 	defer server.Close()

// 	input := struct {
// 		Name     string `json:"name"`
// 		Email    string `json:"email"`
// 		Password string `json:"password"`
// 	}{
// 		Name:     "John Doe",
// 		Email:    "john@example.com",
// 		Password: "password",
// 	}

// 	jsonInput, err := json.Marshal(input)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/register", bytes.NewBuffer(jsonInput))
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req.Header.Set("Content-Type", "application/json")

// 	res, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode != http.StatusCreated {
// 		t.Errorf("expected status %d; got %d", http.StatusCreated, res.StatusCode)
// 	}

// 	var envelope struct {
// 		User *data.User `json:"user"`
// 	}

// 	err = json.NewDecoder(res.Body).Decode(&envelope)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if envelope.User == nil {
// 		t.Errorf("expected user in response; got nil")
// 	}
// }

// func TestServe(t *testing.T) {
// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")

// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	server := &http.Server{
// 		Addr:    fmt.Sprintf(":%d", app.config.port),
// 		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
// 	}

// 	go func() {
// 		if err := app.serve(); err != nil {
// 			t.Errorf("serve returned an error: %v", err)
// 		}
// 	}()
// 	// Wait for the server to start.
// 	time.Sleep(time.Second)

// 	// Send a request to the server.
// 	movie := &data.Movie{
// 		Title:   "Movie 1",
// 		Year:    1999,
// 		Runtime: 64,
// 		Genres:  []string{"Drama", "Sci-Fi"},
// 	}
// 	jsonMovie, err := json.Marshal(movie)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/movies", bytes.NewBuffer(jsonMovie))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	res, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer res.Body.Close()

// 	// Verify the response.
// 	if res.StatusCode != http.StatusOK {
// 		t.Errorf("unexpected status code: got %v, want %v", res.StatusCode, http.StatusOK)
// 	}
// 	var envelope struct {
// 		Movie *data.Movie `json:"movie"`
// 	}
// 	err = json.NewDecoder(res.Body).Decode(&envelope)
// 	if err != nil {
// 		t.Errorf("could not decode response: %v", err)
// 	}
// 	if envelope.Movie == nil {
// 		t.Error("movie is nil")
// 	}

// 	// Shut down the server.
// 	// if err := app.shutdown(); err != nil {
// 	// 	t.Errorf("shutdown returned an error: %v", err)
// 	// }

// 	if err := server.Shutdown(context.Background()); err != nil {
// 		t.Errorf("shutdown returned an error: %v", err)
// 	}
// }

func TestListMoviesHandler(t *testing.T) {
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

	// Insert some test data into the database
	movie1 := &data.Movie{
		Title:   "Movie 1",
		Year:    1999,
		Runtime: 64,
		Genres:  []string{"Drama", "Sci-Fi"},
	}
	err = app.models.Movies.Insert(movie1)
	if err != nil {
		t.Fatal(err)
	}

	movie2 := &data.Movie{
		Title:   "Movie 2",
		Year:    2005,
		Runtime: 92,
		Genres:  []string{"Action", "Adventure"},
	}
	err = app.models.Movies.Insert(movie2)
	if err != nil {
		t.Fatal(err)
	}

	// Make a request to the API to list movies
	res, err := http.Get(server.URL + "/v1/movies")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// Check that the response status code is OK
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}

	// Decode the response JSON into a slice of movies
	var envelope struct {
		Movies []*data.Movie `json:"movies"`
	}
	err = json.NewDecoder(res.Body).Decode(&envelope)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the response contains the expected movies
	if len(envelope.Movies) != 1 {
		t.Fatalf("expected 1 movie; got %d", len(envelope.Movies))
	}

	if envelope.Movies[0].Title != movie1.Title {
		t.Errorf("expected movie title %q; got %q", movie1.Title, envelope.Movies[0].Title)
	}
}

// func TestListMoviesHandlerIntegration(t *testing.T) {
// 	// Create a new application instance
// 	app := &application{}

// 	// Create a new test server instance
// 	ts := httptest.NewServer(http.HandlerFunc(app.listMoviesHandler))
// 	defer ts.Close()

// 	// Define the test cases
// 	testCases := []struct {
// 		name               string
// 		queryString        string
// 		expectedStatusCode int
// 		expectedJSON       string
// 	}{
// 		{
// 			name:               "List all movies with no filters",
// 			queryString:        "",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": []}`,
// 		},
// 		{
// 			name:               "List movies with title filter",
// 			queryString:        "?title=batman",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": [{"id": 1, "title": "Batman", "year": 1989}]}`,
// 		},
// 		{
// 			name:               "List movies with genre filter",
// 			queryString:        "?genres=action",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": [{"id": 1, "title": "Batman", "year": 1989}]}`,
// 		},
// 		// Add more test cases here as needed
// 	}

// 	// Run the test cases
// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Create a new request with the query string
// 			req, err := http.NewRequest("GET", ts.URL+tc.queryString, nil)
// 			if err != nil {
// 				t.Fatalf("could not create request: %v", err)
// 			}

// 			// Send the request and get the response
// 			res, err := http.DefaultClient.Do(req)
// 			if err != nil {
// 				t.Fatalf("could not send request: %v", err)
// 			}
// 			defer res.Body.Close()

// 			// Check the response status code
// 			if res.StatusCode != tc.expectedStatusCode {
// 				t.Errorf("unexpected status code: got %v, want %v", res.StatusCode, tc.expectedStatusCode)
// 			}

// 			// Parse the response JSON
// 			var payload envelope
// 			err = json.NewDecoder(res.Body).Decode(&payload)
// 			if err != nil {
// 				t.Fatalf("could not decode response JSON: %v", err)
// 			}

// 			// Convert the response payload to JSON
// 			payloadJSON, err := json.Marshal(payload)
// 			if err != nil {
// 				t.Fatalf("could not encode payload JSON: %v", err)
// 			}

// 			// Check the response payload
// 			if string(payloadJSON) != tc.expectedJSON {
// 				t.Errorf("unexpected payload: got %v, want %v", string(payloadJSON), tc.expectedJSON)
// 			}
// 		})
// 	}
// }

// func TestListMoviesHandlerIntegration(t *testing.T) {
// 	// Create a new application instance
// 	app := &application{}

// 	// Create a new test server instance
// 	ts := httptest.NewServer(http.HandlerFunc(app.listMoviesHandler))
// 	defer ts.Close()

// 	// Define the test cases
// 	testCases := []struct {
// 		name               string
// 		queryString        string
// 		expectedStatusCode int
// 		expectedJSON       string
// 	}{
// 		{
// 			name:               "List all movies with no filters",
// 			queryString:        "",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": []}`,
// 		},
// 		{
// 			name:               "List movies with title filter",
// 			queryString:        "?title=batman",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": [{"id": 1, "title": "Batman", "year": 1989}]}`,
// 		},
// 		{
// 			name:               "List movies with genre filter",
// 			queryString:        "?genres=action",
// 			expectedStatusCode: http.StatusOK,
// 			expectedJSON:       `{"movies": [{"id": 1, "title": "Batman", "year": 1989}]}`,
// 		},
// 		// Add more test cases here as needed
// 	}

// 	// Run the test cases
// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Create a new request with the query string
// 			req, err := http.NewRequest("GET", ts.URL+tc.queryString, nil)
// 			if err != nil {
// 				t.Fatalf("could not create request: %v", err)
// 			}

// 			// Send the request and get the response
// 			res, err := http.DefaultClient.Do(req)
// 			if err != nil {
// 				t.Fatalf("could not send request: %v", err)
// 			}
// 			defer res.Body.Close()

// 			// Check the response status code
// 			if res.StatusCode != tc.expectedStatusCode {
// 				t.Errorf("unexpected status code: got %v, want %v", res.StatusCode, tc.expectedStatusCode)
// 			}

// 			// Parse the response JSON
// 			var payload envelope
// 			err = json.NewDecoder(res.Body).Decode(&payload)
// 			if err != nil {
// 				t.Fatalf("could not decode response JSON: %v", err)
// 			}

// 			// Check the response payload
// 			if string(payloadJSON) != tc.expectedJSON {
// 				t.Errorf("unexpected payload: got %v, want %v", string(payloadJSON), tc.expectedJSON)
// 			}
// 		})
// 	}
// }

// func TestListMoviesHandler(t *testing.T) {
// 	Start()

// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")
// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	// Initialize a new instance of the application.
// 	// app := newTestApplication(t)

// 	// Define a slice of test movies.
// 	movies := []*data.Movie{
// 		{
// 			Title:   "Memento",
// 			Year:    2000,
// 			Runtime: 113,
// 			Genres:  []string{"Mystery", "Thriller"},
// 		},
// 		{
// 			Title:   "Inception",
// 			Year:    2010,
// 			Runtime: 148,
// 			Genres:  []string{"Action", "Adventure", "Sci-Fi"},
// 		},
// 	}

// 	// Insert the test movies into the database.
// 	for _, movie := range movies {
// 		err := app.models.Movies.Insert(movie)
// 		require.NoError(t, err)
// 	}

// 	tests := []struct {
// 		name            string
// 		url             string
// 		expectedStatus  int
// 		expectedHeaders map[string]string
// 		expectedBody    string
// 	}{
// 		{
// 			name:           "OK",
// 			url:            "/v1/movies?genres=Action&sort=title",
// 			expectedStatus: http.StatusOK,
// 			expectedHeaders: map[string]string{
// 				"Content-Type": "application/json; charset=utf-8",
// 			},
// 			expectedBody: `{
// 				"movies": [
// 					{
// 						"title": "Inception",
// 						"year": 2010,
// 						"runtime": 148,
// 						"genres": ["Action", "Adventure", "Sci-Fi"]
// 					}
// 				]
// 			}`,
// 		},
// 		{
// 			name:           "BadRequestInvalidPage",
// 			url:            "/v1/movies?page=invalid",
// 			expectedStatus: http.StatusBadRequest,
// 			expectedHeaders: map[string]string{
// 				"Content-Type": "application/json; charset=utf-8",
// 			},
// 			expectedBody: `{
// 				"error": "invalid query parameter: page"
// 			}`,
// 		},
// 	}

// 	// Create a new test server
//     ts := httptest.NewServer(http.HandlerFunc(app.listMoviesHandler))
//     defer ts.Close()

//     // Send a GET request to the test server with the appropriate query parameters
//     req, err := http.NewRequest("GET", ts.URL+"?title=hello&genres=action,drama&page=1&page_size=10&sort=title", nil)
//     if err != nil {
//         t.Fatalf("failed to create request: %v", err)
//     }

//     // Send the request and check the response status code and body
//     res, err := http.DefaultClient.Do(req)
//     if err != nil {
//         t.Fatalf("failed to send request: %v", err)
//     }

//     defer res.Body.Close()

//     if res.StatusCode != http.StatusOK {
//         t.Errorf("expected status code %d but got %d", http.StatusOK, res.StatusCode)
//     }

//     // Parse the response body into a struct
//     var envelope struct {
//         Movies []data.Movie `json:"movies"`
//     }

//     err = json.NewDecoder(res.Body).Decode(&envelope)
//     if err != nil {
//         t.Fatalf("failed to decode response: %v", err)
//     }

//     // Check that the response body contains the expected movie data
//     if len(envelope.Movies) != 2 {
//         t.Errorf("expected 2 movies but got %d", len(envelope.Movies))
//     }

// 	// Loop over the test cases.
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Create a new GET request with the given URL.
// 			// req, err := http.NewRequest("GET", app.server.URL+tt.url, nil)
// 			req, err := http.NewRequest("GET", app.server.URL+tt.url, nil)
// 			require.NoError(t, err)

// 			// Set the request headers.
// 			req.Header.Set("Accept", "application/json")

// 			// Make the request to the application.
// 			res, err := app.client.Do(req)
// 			require.NoError(t, err)
// 			defer res.Body.Close()

// 			// Check that the response status code matches the expected value.
// 			require.Equal(t, tt.expectedStatus, res.StatusCode)

// 			// Check the response headers.
// 			for key, value := range tt.expectedHeaders {
// 				require.Equal(t, value, res.Header.Get(key))
// 			}

// 			// Read the response body.
// 			body, err := ioutil.ReadAll(res.Body)
// 			require.NoError(t, err)

// 			// Remove any leading/trailing white space and new lines from the response body.
// 			body = bytes.TrimSpace(body)
// 			body = bytes.ReplaceAll(body, []byte("\n"), []byte(""))

// 			// Check that the response body matches the expected value.
// 			require.JSONEq(t, tt.expectedBody, string(body))
// 		})
// 	}
// }

// func TestListMoviesHandler(t *testing.T) {
// 	Start()

// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")
// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	// app := newTestApplication()
// 	// defer app.models.DB.Close()

// 	// Insert some sample movies into the database
// 	// movie1 := &data.Movie{
// 	// 	Title:   "Movie 1",
// 	// 	Year:    2021,
// 	// 	Runtime: 120,
// 	// 	Genres:  []string{"Action", "Drama"},
// 	// }
// 	// err = app.models.Movies.Insert(movie1)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	// movie2 := &data.Movie{
// 	// 	Title:   "Movie 2",
// 	// 	Year:    2022,
// 	// 	Runtime: 90,
// 	// 	Genres:  []string{"Comedy"},
// 	// }
// 	// err = app.models.Movies.Insert(movie2)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	// // Create a test request
// 	// url := "/v1/movies?title=Movie&genres=Action,Drama&sort=title&page=1&page_size=1"
// 	// req, err := http.NewRequest("GET", url, nil)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	// rr := httptest.NewRecorder()
// 	// handler := http.HandlerFunc(app.listMoviesHandler)
// 	// handler.ServeHTTP(rr, req)

// 	// // Check the response status code
// 	// if status := rr.Code; status != http.StatusOK {
// 	// 	t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
// 	// }

// 	// // Check the response body
// 	// var envelope struct {
// 	// 	Movies []*data.Movie `json:"movies"`
// 	// }
// 	// err = json.Unmarshal(rr.Body.Bytes(), &envelope)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	// if len(envelope.Movies) != 1 {
// 	// 	t.Errorf("handler returned wrong number of movies: got %v want %v", len(envelope.Movies), 1)
// 	// }

// 	// if envelope.Movies[0].Title != "Movie 1" {
// 	// 	t.Errorf("handler returned wrong movie title: got %v want %v", envelope.Movies[0].Title, "Movie 1")
// 	// }

// 	tests := []struct {
// 		name       string
// 		url        string
// 		wantStatus int
// 		wantBody   []byte
// 	}{
// 		{
// 			name:       "OK",
// 			url:        "/movies?title=terminator&genres=action,sci-fi&page=1&page_size=20",
// 			wantStatus: http.StatusOK,
// 			wantBody:   []byte(`{"movies":[{"id":1,"title":"Terminator","year":1984,"runtime":107,"genres":["Action","Sci-Fi"],"version":1}]}`),
// 		},
// 		{
// 			name:       "Bad Request - Invalid Page",
// 			url:        "/movies?page=invalid",
// 			wantStatus: http.StatusBadRequest,
// 			wantBody:   []byte(`{"error":"page must be a positive integer"}`),
// 		},
// 		// add more test cases as necessary
// 	}

// 	// run tests
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// make request to server
// 			rr := httptest.NewRecorder()
// 			req, err := http.NewRequest("GET", tt.url, nil)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			app.routes().ServeHTTP(rr, req)

// 			// check response status code
// 			if status := rr.Code; status != tt.wantStatus {
// 				t.Errorf("handler returned wrong status code: got %v want %v",
// 					status, tt.wantStatus)
// 			}

// 			// check response body
// 			if !bytes.Contains(rr.Body.Bytes(), tt.wantBody) {
// 				t.Errorf("handler returned unexpected body: got %q want %q",
// 					rr.Body.String(), tt.wantBody)
// 			}
// 		})
// 	}
// }

// func TestListMoviesHandler(t *testing.T) {
// 	Start()

// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")
// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	// Setup
// 	// defer app.models.Movies.Truncate()

// 	// Insert test data
// 	movies := []*data.Movie{
// 		{
// 			Title:   "Test Movie 1",
// 			Genres:  []string{"Action", "Adventure"},
// 			Year:    2021,
// 			Runtime: 120,
// 		},
// 		{
// 			Title:   "Test Movie 2",
// 			Genres:  []string{"Drama"},
// 			Year:    2022,
// 			Runtime: 90,
// 		},
// 		{
// 			Title:   "Test Movie 3",
// 			Genres:  []string{"Comedy"},
// 			Year:    2023,
// 			Runtime: 100,
// 		},
// 	}
// 	for _, m := range movies {
// 		err := app.models.Movies.Insert(m)
// 		require.NoError(t, err)
// 	}

// 	tests := []struct {
// 		name           string
// 		url            string
// 		expectedStatus int
// 		expectedMovies []*data.Movie
// 	}{
// 		{
// 			name:           "Success with default values",
// 			url:            "/v1/movies",
// 			expectedStatus: http.StatusOK,
// 			expectedMovies: movies,
// 		},
// 		{
// 			name:           "Success with title filter",
// 			url:            "/v1/movies?title=Test%20Movie%201",
// 			expectedStatus: http.StatusOK,
// 			expectedMovies: movies[:1],
// 		},
// 		{
// 			name:           "Success with genres filter",
// 			url:            "/v1/movies?genres=Action",
// 			expectedStatus: http.StatusOK,
// 			expectedMovies: movies[:1],
// 		},
// 		{
// 			name:           "Success with paging",
// 			url:            "/v1/movies?page=2&page_size=1",
// 			expectedStatus: http.StatusOK,
// 			expectedMovies: movies[1:2],
// 		},
// 		{
// 			name:           "Success with sorting",
// 			url:            "/v1/movies?sort=-year",
// 			expectedStatus: http.StatusOK,
// 			expectedMovies: []*data.Movie{movies[2], movies[1], movies[0]},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Send request
// 			status, headers, body := app.makeRequest("GET", tt.url, nil)

// 			// Check response status code
// 			require.Equal(t, tt.expectedStatus, status)

// 			// Check response headers
// 			require.Contains(t, headers.Get("Content-Type"), "application/json")

// 			// Check response body
// 			var data envelope
// 			err := json.Unmarshal(body, &data)
// 			require.NoError(t, err)
// 			require.NotNil(t, data)

// 			var gotMovies []*data.Movie
// 			err = json.Unmarshal(data["movies"], &gotMovies)
// 			require.NoError(t, err)

// 			require.Equal(t, tt.expectedMovies, gotMovies)
// 		})
// 	}
// }

// func TestCreateMovieHandler(t *testing.T) {
// 	Start()

// 	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

// 	db, err := OpenDB(cfg)
// 	if err != nil {
// 		logger.Fatalf("Connection failed. Error is: %s", err)
// 	}

// 	defer db.Close()
// 	logger.Printf("database connection pool established")
// 	app := &application{
// 		config: cfg,
// 		logger: logger,
// 		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
// 		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
// 	}

// 	// Create a new movie
// 	movie := &data.Movie{
// 		Title:   "The Godfather",
// 		Year:    1972,
// 		Runtime: 175,
// 		Genres:  []string{"Crime", "Drama"},
// 	}

// 	// Encode the movie to JSON
// 	jsonBytes, err := json.Marshal(movie)
// 	require.NoError(t, err)

// 	// Create a new HTTP request
// 	req, err := http.NewRequest("POST", "/v1/movies", bytes.NewReader(jsonBytes))
// 	require.NoError(t, err)

// 	// Set the content type header
// 	req.Header.Set("Content-Type", "application/json")

// 	// Make the request and get the response
// 	recorder := httptest.NewRecorder()
// 	// app.router.ServeHTTP(recorder, req)
// 	app.routes().ServeHTTP(recorder, req)

// 	// Check the status code
// 	require.Equal(t, http.StatusCreated, recorder.Code)

// 	// Check the location header
// 	locationHeader := recorder.Header().Get("Location")
// 	require.NotEmpty(t, locationHeader)

// 	// Check the response body
// 	var responseBody map[string]interface{}
// 	err = json.Unmarshal(recorder.Body.Bytes(), &responseBody)
// 	require.NoError(t, err)

// 	// Check the movie in the response
// 	responseMovie, ok := responseBody["movie"].(map[string]interface{})
// 	require.True(t, ok)

// 	require.Equal(t, movie.Title, responseMovie["title"])
// 	require.Equal(t, movie.Year, int32(responseMovie["year"].(float64)))
// 	require.Equal(t, movie.Runtime, int32(responseMovie["runtime"].(float64)))
// 	require.ElementsMatch(t, movie.Genres, responseMovie["genres"].([]interface{}))
// }
