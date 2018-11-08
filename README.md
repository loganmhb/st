# st

Minimal link shortener intended for limited personal use. Stores links in a SQLite db locally.

There are no metrics and it doesn't track anything. You can't manage or delete the links without accessing the underlying db. There is no authentication or SSL support. If you are running it on the public internet, run it behind nginx or similar, and protect the `/add` route with basic auth.

Usage:

```
st -port 8080 -db /var/st/links.db
```

Starts an HTTP server. Visit /add to add a link.
