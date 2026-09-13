package handlers

import (
	"log/slog"
	"net/http"

	"github.com/grvbrk/nazrein_server/internal/store"
)

type UserHandler struct {
	UserStore store.UserStore
	Logger    *slog.Logger
}

func NewUserHandler(userStore store.UserStore, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		UserStore: userStore,
		Logger:    logger,
	}
}

func (uh *UserHandler) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {

}
