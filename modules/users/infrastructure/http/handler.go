// Package http provides HTTP handlers for the users module.
// Handlers translate HTTP requests into commands/queries and format responses.
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/rai/clean-modularmonolith-go/modules/users/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/queries"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// Handler handles HTTP requests for the users module.
type Handler struct {
	createUser *commands.CreateUserHandler
	updateUser *commands.UpdateUserHandler
	deleteUser *commands.DeleteUserHandler
	getUser    *queries.GetUserHandler
	listUsers  *queries.ListUsersHandler
}

// RegisterRoutes registers the users module routes to the given mux.
func RegisterRoutes(
	mux *http.ServeMux,
	createUser *commands.CreateUserHandler,
	updateUser *commands.UpdateUserHandler,
	deleteUser *commands.DeleteUserHandler,
	getUser *queries.GetUserHandler,
	listUsers *queries.ListUsersHandler,
) {
	h := &Handler{
		createUser: createUser,
		updateUser: updateUser,
		deleteUser: deleteUser,
		getUser:    getUser,
		listUsers:  listUsers,
	}

	mux.HandleFunc("GET /users", h.handleListUsers)
	mux.HandleFunc("POST /users", h.handleCreateUser)
	mux.HandleFunc("GET /users/{id}", h.handleGetUser)
	mux.HandleFunc("PUT /users/{id}", h.handleUpdateUser)
	mux.HandleFunc("DELETE /users/{id}", h.handleDeleteUser)
}

// Request/Response DTOs

type createUserRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type createUserResponse struct {
	ID string `json:"id"`
}

type updateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handlers

func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := commands.CreateUserCommand{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	id, err := h.createUser.Handle(r.Context(), cmd)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createUserResponse{ID: id})
}

func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "user ID is required")
		return
	}

	query := queries.GetUserQuery{UserID: id}
	user, err := h.getUser.Handle(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "user ID is required")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := commands.UpdateUserCommand{
		UserID:    id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	if err := h.updateUser.Handle(r.Context(), cmd); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "user ID is required")
		return
	}

	cmd := commands.DeleteUserCommand{UserID: id}
	if err := h.deleteUser.Handle(r.Context(), cmd); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	query := queries.ListUsersQuery{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.listUsers.Handle(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Helper functions

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrEmailExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUserDeleted):
		writeError(w, http.StatusGone, err.Error())
	case errors.Is(err, domain.ErrEmailInvalid),
		errors.Is(err, domain.ErrEmailRequired),
		errors.Is(err, domain.ErrFirstNameRequired),
		errors.Is(err, domain.ErrLastNameRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
