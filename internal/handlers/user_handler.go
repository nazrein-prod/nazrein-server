package handlers

import (
	"log"
	"net/http"

	"github.com/grvbrk/nazrein_server/internal/store"
)

type UserHandler struct {
	UserStore store.UserStore
	Logger    *log.Logger
}

func NewUserHandler(userStore store.UserStore, logger *log.Logger) *UserHandler {
	return &UserHandler{
		UserStore: userStore,
		Logger:    logger,
	}
}

func (uh *UserHandler) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {

}
