package middleware

import "net/http"

type Handler func(http.ResponseWriter, *http.Request, RequestParameters)