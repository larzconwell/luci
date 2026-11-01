package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"text/template"
	"uuid"

	"github.com/larzconwell/luci"
	"github.com/unrolled/render"
)

const (
	Index          = "index"
	CreatePostGET  = "create_post_get"
	CreatePostPOST = "create_post_post"
	UpdatePostGET  = "update_post_get"
	UpdatePostPOST = "update_post_post"
	GetPost        = "get_post"
	GetAllPosts    = "get_all_posts"
	Status         = "status"
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

func (app *Application) ListenAndServe(ctx context.Context) error {
	err := app.server.ListenAndServe(ctx)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (app *Application) Routes() []luci.Route {
	return []luci.Route{
		{Name: Index, Pattern: "/", HandlerFunc: app.Index},
		{Name: CreatePostGET, Method: http.MethodGet, Pattern: "/posts/create", HandlerFunc: app.CreatePostGET},
		{Name: CreatePostPOST, Method: http.MethodPost, Pattern: "/posts/create", HandlerFunc: app.CreatePostPOST},
		{Name: UpdatePostGET, Method: http.MethodGet, Pattern: "/posts/{id:[0-9a-zA-Z_]+}/update", HandlerFunc: app.UpdatePostGET},
		{Name: UpdatePostPOST, Method: http.MethodPost, Pattern: "/posts/{id:[0-9a-zA-Z_]+}/update", HandlerFunc: app.UpdatePostPOST},
		{Name: GetPost, Pattern: "/posts/{id:[0-9a-zA-Z_]+}", HandlerFunc: app.GetPost},
		{Name: GetAllPosts, Pattern: "/posts", HandlerFunc: app.GetAllPosts},
		{Name: Status, Pattern: "/status", HandlerFunc: app.Status},
	}
}

func (app *Application) Middlewares() luci.Middlewares {
	return nil
}

func (app *Application) Error(rw http.ResponseWriter, req *http.Request, status int, err error) {
	createPostRoute, _ := app.server.Route(CreatePostGET)
	getAllPostsRoute, _ := app.server.Route(GetAllPosts)

	if errors.Is(err, luci.ErrNotFound) {
		err = errors.New("not found")
	} else if errors.Is(err, luci.ErrMethodNotAllowed) {
		err = errors.New("method not allowed")
	}

	rerr := app.render.HTML(rw, status, "error", TemplateData{
		Title: "Error",
		Links: [][2]string{
			{createPostRoute.Pattern, "Create"},
			{getAllPostsRoute.Pattern, "Posts"},
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
	getAllPostsRoute, _ := app.server.Route(GetAllPosts)

	http.Redirect(rw, req, getAllPostsRoute.Pattern, http.StatusFound)
}

func (app *Application) CreatePostGET(rw http.ResponseWriter, req *http.Request) {
	getAllPostsRoute, _ := app.server.Route(GetAllPosts)

	app.respondHTML(rw, req, "create_post", TemplateData{
		Title: "Create",
		Links: [][2]string{
			{getAllPostsRoute.Pattern, "Posts"},
		},
	})
}

func (app *Application) CreatePostPOST(rw http.ResponseWriter, req *http.Request) {
	getPostRoute, _ := app.server.Route(GetPost)
	id := uuid.NewV7().String()

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

	getPost, err := getPostRoute.Path(id)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("failed to create post"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("author", author),
		).Error("failed to build get post link")

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
			slog.Any("error", err),
			slog.String("author", author),
			slog.String("id", post.ID),
		).Error("failed to create post")

		return
	}

	http.Redirect(rw, req, getPost, http.StatusSeeOther)
}

func (app *Application) UpdatePostGET(rw http.ResponseWriter, req *http.Request) {
	createPostRoute, _ := app.server.Route(CreatePostGET)
	getPostRoute, _ := app.server.Route(GetPost)
	getAllPostsRoute, _ := app.server.Route(GetAllPosts)
	id := luci.Var(req, "id")

	getPost, err := getPostRoute.Path(id)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("internal server error"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("id", id),
		).Error("failed to build get post link")

		return
	}

	post, found := app.db.Get(id)
	if !found {
		app.Error(rw, req, http.StatusNotFound, errors.New("post not found"))
		return
	}

	app.respondHTML(rw, req, "update_post", TemplateData{
		Title: "Update",
		Links: [][2]string{
			{createPostRoute.Pattern, "Create"},
			{getPost, "Post"},
			{getAllPostsRoute.Pattern, "Posts"},
		},
		Data: post,
	})
}

func (app *Application) UpdatePostPOST(rw http.ResponseWriter, req *http.Request) {
	getPostRoute, _ := app.server.Route(GetPost)
	id := luci.Var(req, "id")

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

	getPost, err := getPostRoute.Path(id)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("failed to update post"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("id", id),
		).Error("failed to build get post link")

		return
	}

	updated := app.db.Update(id, PostUpdate{
		Title: new(title),
		Link:  new(link),
	})
	if !updated {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("failed to update post"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("id", id),
		).Error("failed to update post")

		return
	}

	// luci.Route.Path calls url.PathEscape on provided route values.
	//gosec:disable G710
	http.Redirect(rw, req, getPost, http.StatusSeeOther)
}

func (app *Application) GetPost(rw http.ResponseWriter, req *http.Request) {
	updatePostRoute, _ := app.server.Route(UpdatePostGET)
	createPostRoute, _ := app.server.Route(CreatePostGET)
	getAllPostsRoute, _ := app.server.Route(GetAllPosts)
	id := luci.Var(req, "id")

	updatePost, err := updatePostRoute.Path(id)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, errors.New("internal server error"))

		luci.Logger(req).With(
			slog.Any("error", err),
			slog.String("id", id),
		).Error("failed to build update post link")

		return
	}

	post, found := app.db.Get(id)
	if !found {
		app.Error(rw, req, http.StatusNotFound, errors.New("post not found"))
		return
	}

	app.respondHTML(rw, req, "get_post", TemplateData{
		Title: post.Title,
		Links: [][2]string{
			{updatePost, "Update"},
			{createPostRoute.Pattern, "Create"},
			{getAllPostsRoute.Pattern, "Posts"},
		},
		Data: post,
	})
}

func (app *Application) GetAllPosts(rw http.ResponseWriter, req *http.Request) {
	createPostRoute, _ := app.server.Route(CreatePostGET)
	posts := app.db.GetAll()

	app.respondHTML(rw, req, "get_all_posts", TemplateData{
		Title: "Posts",
		Links: [][2]string{
			{createPostRoute.Pattern, "Create"},
		},
		Data: posts,
	})
}

func (app *Application) Status(rw http.ResponseWriter, req *http.Request) {
	app.respondJSON(rw, req, map[string]string{
		"server": "healthy",
	})
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
