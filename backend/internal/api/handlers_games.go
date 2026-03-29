package api

import (
	"errors"
	"net/http"
	"path"
)

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cfg.Get().Games)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.AddGame(req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "added game: "+game.Name)
		writeJSON(w, http.StatusCreated, game)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGameByID(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	if id == "" || id == "games" {
		writeErr(w, http.StatusBadRequest, errors.New("missing game id"))
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.UpdateGame(id, req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "updated game: "+game.Name)
		writeJSON(w, http.StatusOK, game)
	case http.MethodDelete:
		if err := s.cfg.DeleteGame(id); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "deleted game id: "+id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
