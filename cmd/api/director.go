package main

import (
	// "errors"
	"fmt"
	"net/http"

	"github.com/shynggys9219/greenlight/internal/data"
)

func (app *application) createDirectorHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string   `json:"name"`
		Surname string   `json:"surname"`
		Awards  []string `json:"awards"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	director := &data.Director{
		Name:    input.Name,
		Surname: input.Surname,
		Awards:  input.Awards,
	}

	// err = app.models.Director.Insert(director)
	err = app.models.Directors.Insert(director)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/directors/%d", director.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"director": director}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// func (app *application) listDirectorsHandler(w http.ResponseWriter, r *http.Request) {
// 	var input struct {
// 		Name   string
// 		Awards []string
// 		data.Filters
// 	}

// 	qs := r.URL.Query()

// 	input.Name = app.readString(qs, "name", "")
// 	// input.Surname = app.readString(qs, "surname", "")
// 	input.Awards = app.readCSV(qs, "awards", []string{})

// 	input.Filters.Page = app.readInt(qs, "page", 1)
// 	input.Filters.PageSize = app.readInt(qs, "page_size", 20)
// 	input.Filters.Sort = app.readString(qs, "sort", "id")

// 	input.Filters.SortSafelist = []string{"id", "name", "surname", "awards", "-id", "-name", "-surname", "-awards"}

// 	directors, err := app.models.Directors.GetAll(input.Name, input.Awards, input.Filters)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	err = app.writeJSON(w, http.StatusOK, envelope{"directors": directors}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}
// }

func (app *application) searchByName(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name   string
		Awards []string
		data.Filters
	}

	qs := r.URL.Query()
	name, err := app.readNameParam(r)
	input.Name = name
	// input.Surname = app.readString(qs, "surname", "")
	input.Awards = app.readCSV(qs, "awards", []string{})
	fmt.Println(input)
	input.Filters.Page = app.readInt(qs, "page", 1)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20)
	input.Filters.Sort = app.readString(qs, "sort", "id")

	input.Filters.SortSafelist = []string{"id", "name", "surname", "awards", "-id", "-name", "-surname", "-awards"}

	// directors, err := app.models.Director.GetOneByName(input.Name, input.Filters)
	// if err != nil {
	// 	app.serverErrorResponse(w, r, err)
	// 	return
	// }
	// directors, err := app.models.Directors.GetAll(input.Name, input.Awards, input.Filters)
	directors, err := app.models.Directors.GetOneByName(input.Name, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"directors": directors}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint.
// TO-DO: Change this handler to retrieve data from a real db
// func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
// 	id, err := app.readIDParam(r)
// 	if err != nil {
// 		app.notFoundResponse(w, r)
// 	}

// 	movie, err := app.models.Movies.Get(id)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			app.notFoundResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}
// 	// Encode the struct to JSON and send it as the HTTP response.
// 	// using envelope
// 	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}
// }

// // TO-DO: Erase existing data by id
// func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
// 	id, err := app.readIDParam(r)
// 	if err != nil {
// 		app.notFoundResponse(w, r)
// 		return
// 	}

// 	err = app.models.Movies.Delete(id)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			app.notFoundResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}

// }

// // TO-DO: Update existing movie
// func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
// 	id, err := app.readIDParam(r)
// 	if err != nil {
// 		app.notFoundResponse(w, r)
// 		return
// 	}

// 	movie, err := app.models.Movies.Get(id)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			app.notFoundResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	var input struct {
// 		Title   string   `json:"title"`
// 		Year    int32    `json:"year"`
// 		Runtime int32    `json:"runtime"`
// 		Genres  []string `json:"genres"`
// 	}

// 	err = app.readJSON(w, r, &input)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	movie.Title = input.Title
// 	movie.Year = input.Year
// 	movie.Runtime = input.Runtime
// 	movie.Genres = input.Genres

// 	err = app.models.Movies.Update(movie)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}

// }
