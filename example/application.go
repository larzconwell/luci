package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larzconwell/luci"
)

const (
	Status     = "status"
	ShowUser   = "show_user"
	UpdateUser = "update_user"
)

type Application struct {
	db     *DB
	server *luci.Server
}

func NewApplication(config luci.Config) *Application {
	app := &Application{
		db: NewDB(map[string]*User{
			"abc123": {Key: "abc123", Name: "luci"},
		}),
	}

	app.server = luci.NewServer(config, app)
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
		{Name: Status, Pattern: "/status", HandlerFunc: app.Status},
		{Name: ShowUser, Method: http.MethodGet, Pattern: "/user/{key:[0-9a-zA-Z]+}", HandlerFunc: app.ShowUser},
		{Name: UpdateUser, Method: http.MethodPost, Pattern: "/user/{key:[0-9a-zA-Z]+}/update", HandlerFunc: app.UpdateUser},
	}
}

func (app *Application) Middlewares() []luci.Middleware {
	return nil
}

func (app *Application) Error(rw http.ResponseWriter, req *http.Request, status int, err error) {
	statusRoute, _ := app.server.Route(Status)
	value := map[string]any{
		"error":  err.Error(),
		"status": status,
		"links": map[string]string{
			Status: statusRoute.Pattern,
		},
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)

	encoder := json.NewEncoder(rw)
	encodeErr := encoder.Encode(value)
	if encodeErr != nil && !errors.Is(encodeErr, http.ErrHandlerTimeout) && !errors.Is(encodeErr, context.Canceled) {
		luci.Logger(req).With(
			slog.Any("error", encodeErr),
			slog.Any("source_error", err),
		).Error("Failed to write response")
	}
}

func (app *Application) Respond(rw http.ResponseWriter, req *http.Request, value any) {
	rw.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(rw)
	err := encoder.Encode(value)
	if err != nil && !errors.Is(err, http.ErrHandlerTimeout) && !errors.Is(err, context.Canceled) {
		luci.Logger(req).With(slog.Any("error", err)).Error("Failed to write response")
	}
}

func (app *Application) Status(rw http.ResponseWriter, req *http.Request) {
	app.Respond(rw, req, map[string]string{
		"server": "healthy",
	})
}

func (app *Application) ShowUser(rw http.ResponseWriter, req *http.Request) {
	key := luci.RequestVar(req, "key")

	user, ok := app.db.Get(key)
	if !ok {
		app.Error(rw, req, http.StatusNotFound, errors.New("user not found"))
		return
	}

	updateUserRoute, _ := app.server.Route(UpdateUser)
	updateUserPath, err := updateUserRoute.Path(key)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	app.Respond(rw, req, map[string]any{
		"user": user,
		"links": map[string]string{
			ShowUser:   req.URL.Path,
			UpdateUser: updateUserPath,
		},
	})
}

func (app *Application) UpdateUser(rw http.ResponseWriter, req *http.Request) {
	key := luci.RequestVar(req, "key")

	name := req.FormValue("name")
	if name == "" {
		app.Error(rw, req, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	user := app.db.Update(key, name)

	showUserRoute, _ := app.server.Route(ShowUser)
	showUserPath, err := showUserRoute.Path(key)
	if err != nil {
		app.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	app.Respond(rw, req, map[string]any{
		"user": user,
		"links": map[string]string{
			ShowUser:   showUserPath,
			UpdateUser: req.URL.Path,
		},
	})
}
