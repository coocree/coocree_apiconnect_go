package middleware

import (
	"context"
	"net/http"
	"time"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

//var cookieAccessKeyCtx = &contextKey{"cookies"}

type contextKey struct {
	name string
}

// A stand-in for our database backed user object
type User struct {
	Name    string
	IsAdmin bool
}

// Middleware decodes the share session cookie and packs the session into context
func Session() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("auth-cookie")

			//fmt.Println("auth-cookie--->>>CCCCCC", c, "<<---", r.Cookies())

			// Allow unauthenticated users in
			if err != nil || c == nil {
				//next.ServeHTTP(w, r)
				//return
			}

			/*
				userId, err := validateAndGetUserID(c)
				if err != nil {
					http.Error(w, "Invalid cookie", http.StatusForbidden)
					return
				}
			*/

			// get the user from the database
			//user := getUserByID(db, userId)

			user := User{
				Name:    "Israel",
				IsAdmin: false,
			}

			// put it in context
			ctx := context.WithValue(r.Context(), userCtxKey, user)

			//fmt.Println("USER--->>>AAAAAAAA", user)

			expire := time.Now().AddDate(0, 0, 1)
			cookie := http.Cookie{}
			cookie.Name = "test"
			cookie.Value = "tcookie"
			cookie.Path = "/"
			cookie.Domain = "192.168.1.34"
			cookie.Expires = expire
			cookie.RawExpires = expire.Format(time.UnixDate)
			cookie.MaxAge = 86400
			cookie.Secure = false
			cookie.HttpOnly = true
			//cookie.SameSite =
			cookie.Unparsed = []string{"test=tcookie"}

			//fmt.Println("COOKIE--->>>MMMMMMMMM", cookie)

			http.SetCookie(w, &http.Cookie{
				Name:     "auth-cookie",
				Value:    "sssss",
				HttpOnly: true,
				Path:     "/",
				Domain:   "192.168.1.34",
			})

			//"test", "tcookie", "/", "www.domain.com", expire, expire.Format(time.UnixDate), 86400, true, true, "test=tcookie", []string{"test=tcookie"}

			r.AddCookie(&cookie)
			//io.WriteString(w, "Hello world!")

			// and call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) User {

	//fmt.Println("USER--->>>KKKKKKKKK", userCtxKey)
	//fmt.Println("USER--->>>EEEEEEEEEE", ctx.Value(userCtxKey))
	//x := ctx.Value(userCtxKey)
	//fmt.Println("USER--->>>XXXXXXXXXXXXXX", x, x.(User).Name)

	raw, err := ctx.Value(userCtxKey).(User)
	if err {
		//fmt.Println("USER--->>>FFFFFFFFFFFFFFFF", err)
	}
	return raw
}
