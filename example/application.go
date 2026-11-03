package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"text/template"
	"uuid"

	"github.com/larzconwell/luci"
	"github.com/unrolled/render"
)

const (
	Index  = "index"
	Status = "status"

	PostsGetAll     = "posts_get_all"
	PostsCreateGET  = "posts_create_get"
	PostsCreatePOST = "posts_create_post"
	PostsGet        = "posts_get"
	PostsUpdateGET  = "posts_update_get"
	PostsUpdatePOST = "posts_update_post"
)

//go:embed templates
var templates embed.FS

type TemplateData struct {
	Title string
	Links [][2]string
	Data  any
}

type Application struct {
	db     *DB
	render *render.Render
	server *luci.Server
}

func NewApplication(config luci.Config) *Application {
	app := &Application{
		db: NewDB(),
		render: render.New(render.Options{
			FileSystem:                  render.FS(templates),
			Layout:                      "layout",
			Extensions:                  []string{".html"},
			RenderPartialsWithoutPrefix: true,
			DisableHTTPErrorRendering:   true,
		}),
	}

	app.server = luci.NewServer(config, app)
	app.render.CompileTemplates()

	return app
}

func (app *Application) Routes() []luci.Route {
	return []luci.Route{
		{Name: Index, Method: http.MethodGet, Pattern: "/", HandlerFunc: app.Index},
		{Name: Status, Method: http.MethodGet, Pattern: "/status", HandlerFunc: app.Status},
		{
			Method:  http.MethodGet,
			Pattern: "/posts",
			Routes: []luci.Route{
				{Name: PostsGetAll, Pattern: "/", HandlerFunc: app.PostsGetAll},
				{Name: PostsCreateGET, Pattern: "/create", HandlerFunc: app.PostsCreateGET},
				{Name: PostsCreatePOST, Method: http.MethodPost, Pattern: "/create", HandlerFunc: app.PostsCreatePOST},
				{Name: PostsGet, Pattern: "/{id:[0-9a-zA-Z_-]+}", HandlerFunc: app.PostsGet},
				{Name: PostsUpdateGET, Pattern: "/{id:[0-9a-zA-Z_-]+}/update", HandlerFunc: app.PostsUpdateGET},
				{Name: PostsUpdatePOST, Method: http.MethodPost, Pattern: "/{id:[0-9a-zA-Z_-]+}/update", HandlerFunc: app.PostsUpdatePOST},
			},
		},
	}
}

func (app *Application) Middlewares() luci.Middlewares {
	return nil
}

func (app *Application) Error(rw http.ResponseWriter, req *http.Request, status int, err error) {
	createPostRoute := app.server.MustRoute(PostsCreateGET).MustPath()
	getAllPostsRoute := app.server.MustRoute(PostsGetAll).MustPath()

	if errors.Is(err, luci.ErrNotFound) {
		err = errors.New("not found")
	} else if errors.Is(err, luci.ErrMethodNotAllowed) {
		err = errors.New("method not allowed")
	}

	rerr := app.render.HTML(rw, status, "error", TemplateData{
		Title: "Error",
		Links: [][2]string{
			{createPostRoute, "Create"},
			{getAllPostsRoute, "Posts"},
		},
		Data: struct {
			Status int
			Error  error
		}{
			Status: status,
			Error:  err,
		},
	})
	if rerr != nil && !errors.Is(rerr, http.ErrHandlerTimeout) && !errors.Is(rerr, context.Canceled) {
		luci.Logger(req).With(
			slog.Any("error", rerr),
			slog.Any("source_error", err),
		).Error("failed to write response")
	}
}

func (app *Application) Index(rw http.ResponseWriter, req *http.Request) {
	getAllPostsRoute := app.server.MustRoute(PostsGetAll).MustPath()

	http.Redirect(rw, req, getAllPostsRoute, http.StatusFound)
}

func (app *Application) Status(rw http.ResponseWriter, req *http.Request) {
	app.respondJSON(rw, req, map[string]string{
		"server": "healthy",
	})
}

func (app *Application) PostsGetAll(rw http.ResponseWriter, req *http.Request) {
	createPostRoute := app.server.MustRoute(PostsCreateGET).MustPath()
	posts := app.db.GetAll()

	app.respondHTML(rw, req, "posts_get_all", TemplateData{
		Title: "Posts",
		Links: [][2]string{
			{createPostRoute, "Create"},
		},
		Data: posts,
	})
}

func (app *Application) PostsCreateGET(rw http.ResponseWriter, req *http.Request) {
	getAllPostsRoute := app.server.MustRoute(PostsGetAll).MustPath()

	app.respondHTML(rw, req, "posts_create", TemplateData{
		Title: "Create",
		Links: [][2]string{
			{getAllPostsRoute, "Posts"},
		},
	})
}

func (app *Application) PostsCreatePOST(rw http.ResponseWriter, req *http.Request) {
	id := uuid.NewV7().String()
	getPostRoute := app.server.MustRoute(PostsGet).MustPath(id)

	title := req.FormValue("title")
	if title == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("title is required"))
		return
	}

	link := req.FormValue("link")
	if link == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("link is required"))
		return
	}

	author := req.FormValue("author")
	if author == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("author is required"))
		return
	}

	post := Post{
		ID:     id,
		Title:  title,
		Link:   link,
		Author: author,
	}

	created := app.db.Create(post)
	if !created {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("failed to create post"))

		luci.Logger(req).With(
			slog.String("author", author),
			slog.String("id", post.ID),
		).Error("failed to create post")

		return
	}

	http.Redirect(rw, req, getPostRoute, http.StatusSeeOther)
}

func (app *Application) PostsGet(rw http.ResponseWriter, req *http.Request) {
	id := luci.Var(req, "id")
	updatePostRoute := app.server.MustRoute(PostsUpdateGET).MustPath(id)
	createPostRoute := app.server.MustRoute(PostsCreateGET).MustPath()
	getAllPostsRoute := app.server.MustRoute(PostsGetAll).MustPath()

	post, found := app.db.Get(id)
	if !found {
		app.Error(rw, req, http.StatusNotFound, errors.New("post not found"))
		return
	}

	app.respondHTML(rw, req, "posts_get", TemplateData{
		Title: post.Title,
		Links: [][2]string{
			{updatePostRoute, "Update"},
			{createPostRoute, "Create"},
			{getAllPostsRoute, "Posts"},
		},
		Data: post,
	})
}

func (app *Application) PostsUpdateGET(rw http.ResponseWriter, req *http.Request) {
	id := luci.Var(req, "id")
	createPostRoute := app.server.MustRoute(PostsCreateGET).MustPath()
	getPostRoute := app.server.MustRoute(PostsGet).MustPath(id)
	getAllPostsRoute := app.server.MustRoute(PostsGetAll).MustPath()

	post, found := app.db.Get(id)
	if !found {
		app.Error(rw, req, http.StatusNotFound, errors.New("post not found"))
		return
	}

	app.respondHTML(rw, req, "posts_update", TemplateData{
		Title: "Update",
		Links: [][2]string{
			{createPostRoute, "Create"},
			{getPostRoute, "Post"},
			{getAllPostsRoute, "Posts"},
		},
		Data: post,
	})
}

func (app *Application) PostsUpdatePOST(rw http.ResponseWriter, req *http.Request) {
	id := luci.Var(req, "id")
	getPostRoute := app.server.MustRoute(PostsGet).MustPath(id)

	title := req.FormValue("title")
	if title == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("title is required"))
		return
	}

	link := req.FormValue("link")
	if link == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("link is required"))
		return
	}

	updated := app.db.Update(id, PostUpdate{
		Title: new(title),
		Link:  new(link),
	})
	if !updated {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("failed to update post"))
		luci.Logger(req).With(slog.String("id", id)).Error("failed to update post")

		return
	}

	// luci.Route.Path calls url.PathEscape on provided route values.
	//gosec:disable G710
	http.Redirect(rw, req, getPostRoute, http.StatusSeeOther)
}

func (app *Application) respondJSON(rw http.ResponseWriter, req *http.Request, value any) {
	err := app.render.JSON(rw, http.StatusOK, value)
	if err != nil && !errors.Is(err, http.ErrHandlerTimeout) && !errors.Is(err, context.Canceled) {
		luci.Logger(req).With(
			slog.Any("error", err),
		).Error("failed to write response")
	}
}

func (app *Application) respondHTML(rw http.ResponseWriter, req *http.Request, name string, data TemplateData) {
	var execErr template.ExecError

	err := app.render.HTML(rw, http.StatusOK, name, data)
	switch {
	case errors.Is(err, http.ErrHandlerTimeout), errors.Is(err, context.Canceled):
	case errors.As(err, &execErr):
		app.Error(rw, req, http.StatusInternalServerError, errors.New("internal server error"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("template", name),
			slog.Any("data", data),
		).Error("failed to render template")
	case err != nil:
		luci.Logger(req).With(
			slog.Any("error", err),
		).Error("failed to write response")
	}
}
