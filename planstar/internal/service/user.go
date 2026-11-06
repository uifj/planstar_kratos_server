package service

import (
	"net/http"
	"strconv"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

func (s *Service) profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	item, err := s.user.Profile(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) addresses(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.user.Addresses(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) addAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.AddressInput{
		Name:      httpx.StrAny(body, "name", "consignee"),
		Mobile:    httpx.StrAny(body, "mobile", "phone"),
		Region:    httpx.Str(body, "region"),
		Detail:    httpx.StrAny(body, "detail", "address"),
		Province:  httpx.Str(body, "province"),
		City:      httpx.Str(body, "city"),
		District:  httpx.Str(body, "district"),
		IsDefault: httpx.BoolAny(body, "isDefault", "default"),
	}
	item, err := s.user.AddAddress(r.Context(), userID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) deleteAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	if err := s.user.DeleteAddress(r.Context(), userID, id); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{})
}

func (s *Service) setDefaultAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	if err := s.user.SetDefaultAddress(r.Context(), userID, id); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{})
}
