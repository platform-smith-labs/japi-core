// Package realip sets a request's RemoteAddr from proxy headers. It is a
// dependency-free equivalent of chi's middleware.RealIP: importing chi's
// middleware package registers net/http/pprof handlers on
// http.DefaultServeMux, which japi-core must never do.
package realip

import (
	"net"
	"net/http"
	"strings"
)

// Middleware sets r.RemoteAddr to the first valid IP from True-Client-IP,
// X-Real-IP or X-Forwarded-For (in that order), as chi's RealIP does. Use it
// only behind a proxy that controls those headers.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := fromHeaders(r.Header); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func fromHeaders(h http.Header) string {
	var ip string
	if v := h.Get("True-Client-IP"); v != "" {
		ip = v
	} else if v := h.Get("X-Real-IP"); v != "" {
		ip = v
	} else if v := h.Get("X-Forwarded-For"); v != "" {
		ip, _, _ = strings.Cut(v, ",")
	}
	ip = strings.TrimSpace(ip)
	if ip == "" || net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
