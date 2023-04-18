package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		return
	}

	err = app.models.User.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email addres already exists")
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Token.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.Print(err, nil)
		}
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
// 	var input struct {
// 		TokenPlaintext string `json:"token"`
// 	}
// 	err := app.readJSON(w, r, &input)
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}

// 	v := validator.New()
// 	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
// 		app.failedValidationResponse(w, r, v.Errors)
// 		return
// 	}

// 	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			v.AddError("token", "invalid or expired activation token")
// 			app.failedValidationResponse(w, r, v.Errors)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	user.Activated = true

// 	err = app.models.Users.Update(user)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrEditConflict):
// 			app.editConflictResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}
// }

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the plaintext activation token from the request body.
	var input struct {
		TokenPlaintext string `json:"token"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		// app.badRequestResponse(w, r, err)
		return
	}
	// Validate the plaintext token provided by the client.
	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		// app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Retrieve the details of the user associated with the token using the
	// GetForToken() method (which we will create in a minute). If no matching record
	// is found, then we let the client know that the token they provided is not valid.
	user, err := app.models.User.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			// app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Update the user's activation status.
	user.Activated = true
	// Save the updated user record in our database, checking for any edit conflicts in
	// the same way that we did for our movie records.
	err = app.models.User.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// If everything went successfully, then we delete all activation tokens for the
	// user.
	err = app.models.Token.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send the updated user details to the client in a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createRoleHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID       int64  `json:"id"`
		RoleName string `json:"role_name"`
		UserID   int64  `json:"user_id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	role := &data.Role{
		ID:       input.ID,
		RoleName: input.RoleName,
		UserID:   input.UserID,
	}

	err = app.models.Role.InsertUserRole(role)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/roles/%d", role.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"role": role}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
