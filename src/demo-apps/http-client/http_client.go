// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package main

import (
    "context"
    "flag"
    "fmt"
    "io/ioutil"
    "net"
    "net/http"
    "time"
)

func main() {
    var counter = 0

    //httpHost := os.Getenv("SERVERDOMAIN")
    serverURL, dnsServer := getArgs()
    client := getHttpClient(dnsServer)

    for {
	url_with_counter := fmt.Sprintf("%s?counter=%d", serverURL, counter)
        httpGet(client, url_with_counter)
        time.Sleep(5 * time.Second)
    }
}

func getArgs()  (string, string) {
    var serverName, serverPort, dnsServer string

    flag.StringVar(&serverName, "server", "webserver.demo.com", "HTTP server name")
    flag.StringVar(&serverPort, "port", "32612", "HTTP server port")
    flag.StringVar(&dnsServer, "dns", "192.168.0.199:53", "Custom DNS server with port")
    flag.Parse()

    serverURL := "http://" + serverName + ":" + serverPort
    fmt.Printf("Connecting to %s using DNS %s\n", serverURL, dnsServer)

    return serverURL, dnsServer
}

func getHttpClient(dnsResolverIP string) (*http.Client) {
    var dnsResolverTimeoutMs = 5000 // Timeout (ms) for the DNS resolver (optional)

    dialer := &net.Dialer{
        Resolver: &net.Resolver{
            PreferGo: true,
            Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
                d := net.Dialer{
                  Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
                }
                return d.DialContext(ctx, network, dnsResolverIP)
            },
        },
    }

    dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
        return dialer.DialContext(ctx, network, addr)
    }

    http.DefaultTransport.(*http.Transport).DialContext = dialContext
    // httpClient := &http.Client{}
    return &http.Client{}
}

func httpGet(client *http.Client, url string) {
    resp, err := client.Get(url)
    if err != nil {
        fmt.Printf("unable to connect to http server - %s\n", err.Error())
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("unable to read http server response - %s\n", err.Error())
    }

    fmt.Println(string(body))
}

