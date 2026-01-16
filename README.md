# net-gopher

Package `gopher` provides Gopher client and server implementations.

`Get` makes Gopher requests:

```go
resp, err := gopher.Get("gopher://example.com/")
```

The caller must close the response when finished with it:

```go
resp, err := gopher.Get("gopher://example.com/")
if err != nil {
    // handle error
}
defer resp.Close()
body, err := io.ReadAll(resp.Body)
// ...
```

## Clients

For control over Gopher client settings, create a `Client`:

```go
client := &gopher.Client{
    Timeout: 10 * time.Second
}

resp, err := client.Get("gopher://example.com/")
// ...
```

## Servers

`ListenAndServe` starts a Gopher server with a given address and handler.
The handler is usually `nil`, which means to use `DefaultServeMux`.
`Handle` and `HandleFunc` add handlers to `DefaultServeMux`:

```go
gopher.Handle("/", func(w gopher.ResponseWriter, r *gopher.Request) {
    w.WriteInfo("Welcome to the Gopher server!")
    w.WriteItem(&gopher.Item{Type: gopher.TypeDirectory, Display: "About", Selector: "/about"})
    w.WriteItem(&gopher.Item{Type: gopher.TypeText, Display: "README", Selector: "/readme"})
})

gopher.HandleFunc("/about", func(w gopher.ResponseWriter, r *gopher.Request) {
	w.WriteInfo("This is an example Gopher server")
	w.WriteInfo("Built with net-gopher")
})

gopher.HandleFunc("/readme", func(w gopher.ResponseWriter, r *gopher.Request) {
    fmt.Fprintln(w, "net-gopher: A Gopher protocol library for Go")
})

log.Fatal(gopher.ListenAndServe(":7070", nil))
```
