package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/luisguve/scs/v2"
)

// LoadAndSave provides middleware which automatically loads and saves session
// data for the current request, and communicates the session token to and from
// the client in a cookie.
func (r routes) LoadAndSave(c *fiber.Ctx) error {
	token := c.Cookies(r.sess.Cookie.Name)

	ctx, err := r.sess.Load(c.Context(), token)
	if err != nil {
		log.Printf("Could not load sess with token \"%s\": %s\n", token, err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	ctxKey := r.sess.Key()
	ctxVal := ctx.Value(string(ctxKey))
	c.Context().SetUserValue(string(ctxKey), ctxVal)

	if err = c.Next(); err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	if mpf, _ := c.Request().MultipartForm(); mpf != nil {
		mpf.RemoveAll()
	}

	switch r.sess.Status(c.Context()) {
	case scs.Modified:
		token, expiry, err := r.sess.Commit(c.Context())
		if err != nil {
			log.Printf("Could not commit sess: %v\n", err)
			return c.SendStatus(http.StatusInternalServerError)
		}
		r.writeSessionCookie(c, token, expiry)
	case scs.Destroyed:
		r.writeSessionCookie(c, "", time.Time{})
	}

	return nil
}

func (r routes) writeSessionCookie(c *fiber.Ctx, token string, expiry time.Time) {
	cookie := &fiber.Cookie{
		Name:     r.sess.Cookie.Name,
		Value:    token,
		Path:     r.sess.Cookie.Path,
		Domain:   r.sess.Cookie.Domain,
		Secure:   r.sess.Cookie.Secure,
		HTTPOnly: r.sess.Cookie.HttpOnly,
	}

	if expiry.IsZero() {
		cookie.Expires = time.Unix(1, 0)
		cookie.MaxAge = -1
	} else if r.sess.Cookie.Persist {
		cookie.Expires = time.Unix(expiry.Unix()+1, 0)        // Round up to the nearest second.
		cookie.MaxAge = int(time.Until(expiry).Seconds() + 1) // Round up to the nearest second.
	}

	c.Cookie(cookie)
}
