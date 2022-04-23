// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package main

import (
    "fmt"
    "net/http"
    "strconv"
    "time"
)

const serverEndpoint = ":3333"

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(serverEndpoint, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
    keys, ok := r.URL.Query()["data"]
    if !ok {
       http.Error(w, "No data in request", http.StatusBadRequest)
       return
    }

    inData, err := strconv.Atoi(keys[0])
    if err != nil {
       http.Error(w, "Bad data in request", http.StatusBadRequest)
       return
    }

    t := time.Now()
    fmt.Fprintf(w, "%s data: %d\n", t.Format("2006-01-02 15:04:05"), inData)
}

