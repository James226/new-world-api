package main

import (
	"fmt"
	"net/http"
)

type healthController struct {
}

func (c healthController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Healthy\n")
}
