package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/snippetbox/pkg/forms"
	"github.com/snippetbox/pkg/models"
)

type DataTemplate struct {
	Data []*Home
}

type Home struct {
	Snippet *models.Snippet
	Author  string
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {

	s, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	dataTemplate := &DataTemplate{}
	for _, sd := range s {
		author, err := app.snippets.GetAuthor(sd.UserID)
		if err != nil {
			fmt.Errorf("%s", err)
		}
		home := &Home{sd, author}
		dataTemplate.Data = append(dataTemplate.Data, home)
	}
	td := &templateData{
		Temp: dataTemplate,
	}
	app.render(w, r, "home.page.tmpl", td)
}

func (app *application) showSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get(":id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	s, err := app.snippets.Get(id)
	if err != nil {
		if err == models.ErrNoRecord {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.render(w, r, "show.page.tmpl", &templateData{
		Snippet: s,
	})
}

func (app *application) createSnippetForm(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "create.page.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) createSnippet(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("title", "content", "expires")
	form.MaxLength("title", 100)
	form.PermittedValues("expires", "365", "7", "1")
	userID := app.session.GetInt(r, "authenticatedUserID")
	if !form.Valid() {
		app.render(w, r, "create.page.tmpl", &templateData{Form: form})
		return
	}

	id, err := app.snippets.Insert(form.Get("title"), form.Get("content"), form.Get("expires"), userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Quote successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/%d", id), http.StatusSeeOther)
}

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "signup.page.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("name", "email", "password")
	form.MaxLength("name", 255)
	form.MaxLength("email", 255)
	form.MatchesPattern("email", forms.EmailRX)
	form.MinLength("password", 10)

	if !form.Valid() {
		app.render(w, r, "signup.page.tmpl", &templateData{Form: form})
		return
	}

	err = app.users.Insert(form.Get("name"), form.Get("email"), form.Get("password"))
	if err != nil {
		if err == models.ErrDuplicateEmail {
			form.Errors.Add("email", "Address is already in use")
			app.render(w, r, "signup.page.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r, "flash", "Your signup was successful. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) loginUserForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	id, err := app.users.Authenticate(form.Get("email"), form.Get("password"))
	if err != nil {
		if err == models.ErrInvalidCredentials {
			form.Errors.Add("generic", "Email or Password is incorrect")
			app.render(w, r, "login.page.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r, "authenticatedUserID", id)

	path := app.session.PopString(r, "redirectPathAfterLogin")
	if path != "" {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) logoutUser(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r, "authenticatedUserID")

	app.session.Put(r, "flash", "You have been logged out successfully!")
	http.Redirect(w, r, "/", 303)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "about.page.tmpl", nil)
}

func (app *application) userProfile(w http.ResponseWriter, r *http.Request) {
	userID := app.session.GetInt(r, "authenticatedUserID")

	user, err := app.users.Get(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "profile.page.tmpl", &templateData{
		User: user,
	})
}

func (app *application) changePasswordForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "password.page.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) changePassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("currentPassword", "newPassword", "newPasswordConfirmation")
	form.MinLength("newPassword", 10)
	if form.Get("newPassword") != form.Get("newPasswordConfirmation") {
		form.Errors.Add("newPasswordConfirmation", "Passwords do not match")
	}

	if !form.Valid() {
		app.render(w, r, "password.page.tmpl", &templateData{Form: form})
		return
	}

	userID := app.session.GetInt(r, "authenticatedUserID")

	err = app.users.ChangePassword(userID, form.Get("currentPassword"), form.Get("newPassword"))
	if err != nil {
		if err == models.ErrInvalidCredentials {
			form.Errors.Add("currentPassword", "Current password is incorrect")
			app.render(w, r, "password.page.tmpl", &templateData{Form: form})
		} else if err != nil {
			app.serverError(w, err)
		}
		return
	}
	app.session.Put(r, "flash", "Your password has been updated!")
	http.Redirect(w, r, "/user/profile", 303)

}

func (app *application) mysnippets(w http.ResponseWriter, r *http.Request) {
	s, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	userID := app.session.GetInt(r, "authenticatedUserID")

	snippets := []*models.Snippet{}
	for i, snippet := range s {
		if s[i].UserID == userID {
			snippets = append(snippets, snippet)
		}
	}

	app.render(w, r, "mysnippets.page.tmpl", &templateData{
		Snippets: snippets,
	})
}

func (app *application) editSnippetForm(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "edit.page.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) editSnippet(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get(":id"))

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("title", "content", "expires")
	form.MaxLength("title", 100)
	form.PermittedValues("expires", "365", "7", "1")

	userID := app.session.GetInt(r, "authenticatedUserID")

	if !form.Valid() {
		app.render(w, r, "edit.page.tmpl", &templateData{Form: form})
		return
	}

	// idd, err := strconv.Atoi(r.URL.Query().Get(":idd"))
	// fmt.Println("snippet idd je prvo: ", idd)
	// if err != nil || idd < 1 {
	// 	app.notFound(w)
	// 	return
	// }
	// fmt.Println("idd je :", idd)

	// s, err := app.snippets.Get(idd)
	// fmt.Println("snippet id je posle....: ", s)
	// if err != nil {
	// 	if err == models.ErrNoRecord {
	// 		app.notFound(w)
	// 	} else {
	// 		app.serverError(w, err)
	// 	}
	// 	return
	// }

	//fmt.Println("snippet s je posle....: ", s)
	fmt.Println("snippet id je pre: ", id)
	err = app.snippets.Update(form.Get("title"), form.Get("content"), form.Get("expires"), id)
	fmt.Println("snippet id je: ", id)
	if err != nil {
		app.serverError(w, err)
		return
	}

	//fmt.Println("snippet id je sada opet: ", id)
	fmt.Println("USER je: ", userID)

	s, err := app.snippets.Latest()
	fmt.Println("id je: ", s)
	if err != nil {
		app.serverError(w, err)
		return
	}

	snippets := []*models.Snippet{}
	for i, snippet := range s {
		if s[i].UserID == userID {
			snippets = append(snippets, snippet)
		}
		fmt.Println("snipets je: ", snippets)
	}

	app.session.Put(r, "flash", "Quote successfully edited!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/%d", id), http.StatusSeeOther)

	app.render(w, r, "edit.page.tmpl", &templateData{
		Snippets: snippets,
	})

}

// func (app *application) deleteSnippet(w http.ResponseWriter, r *http.Request) {

// 	userID := app.session.GetInt(r, "authenticatedUserID")

//  }
