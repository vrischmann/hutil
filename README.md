# hutil

# Introduction

hutil is a set of helpers to work with `net/http`. It's limited by design, based on what I need the most in my projects.

# Middleware

The most useful type is `hutil.MiddlewareStack`. It is used to stack middlewares and wrap a [http.Handler](https://pkg.go.dev/net/http#Handler). Look at the documentation [here](https://pkg.go.dev/github.com/vrischmann/hutil/v3#MiddlewareStack) to see how to use it.

We also provide a logging middleware. Look at the documentation [here](https://pkg.go.dev/github.com/vrischmann/hutil/v3#NewLoggingMiddleware) to see how to use it.

# Logging

You can get a logging middlware this way:

```go
logger := zap.NewExample()
m := hutil.NewLoggingMiddleware(logger)
```

# Shift path

This function is useful to build complex routing based on the URL path using only the standard library HTTP package.

It's more involved than using something like `gorilla/mux` but not that much for basic things.

Suppose you have the following routes:

```
/user
/user/feed/<id>
/user/profile/<id>
/search/inventory
/search/company
```

You could build your routing like this:

```go
type userHandler struct {
}

func (h *userHandler) handle(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = hutil.ShiftPath(req.URL.Path)
	switch head {
	case "profile":
		h.profile(w, req)
	case "feed":
		h.feed(w, req)
	}
}

func (h *userHandler) profile(w http.ResponseWriter, req *http.Request) {
	profileID := req.URL.Path
	fmt.Printf("profile, profile id: %s\n", profileID)
}
func (h *userHandler) feed(w http.ResponseWriter, req *http.Request) {
	profileID := req.URL.Path
	fmt.Printf("feed, profile id: %s\n", profileID)
}

type searchHandler struct {
}

func (h *searchHandler) handle(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = hutil.ShiftPath(req.URL.Path)
	switch head {
	case "inventory":
		h.inventorySearch(w, req)
	case "company":
		h.companySearch(w, req)
	}
}

func (h *searchHandler) inventorySearch(w http.ResponseWriter, req *http.Request) {
	fmt.Println("inventory search")
}
func (h *searchHandler) companySearch(w http.ResponseWriter, req *http.Request) {
	fmt.Println("company search")
}

type router struct {
	uh *userHandler
	sh *searchHandler
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = hutil.ShiftPath(req.URL.Path)
	switch head {
	case "user":
		r.uh.handle(w, req)
	case "search":
		r.sh.handle(w, req)
	}
}

func main() {
	r := &router{
		uh: new(userHandler),
		sh: new(searchHandler),
	}
	http.ListenAndServe(":3204", r)
}
```
