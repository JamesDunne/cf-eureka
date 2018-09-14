package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Expected arguments: <routes.json path> argument to load JSON file that describes apps to register in eureka")
		os.Exit(1)
		return
	}

	// Create JSON encoder to write to stdout:
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// Get eureka URL from args:
	eurekaHost := os.Getenv("EUREKA_HOST")
	if eurekaHost == "" {
		fmt.Fprintln(os.Stderr, "WARNING: missing EUREKA_HOST env; assuming 'localhost'")
		eurekaHost = "localhost"
	}
	eurekaPort := os.Getenv("EUREKA_PORT")
	if eurekaPort == "" {
		fmt.Fprintln(os.Stderr, "WARNING: missing EUREKA_PORT env; assuming '8080'")
		eurekaPort = "8080"
	}

	// Get space name from first argument:
	routesFilePath := args[0]

	var inReader io.Reader
	if routesFilePath == "-" {
		inReader = os.Stdin
	} else {
		b, err := ioutil.ReadFile(routesFilePath)
		if err != nil {
			panic(err)
		}

		inReader = bytes.NewReader(b)
	}

	var routesMap map[string]string
	if err := json.NewDecoder(inReader).Decode(&routesMap); err != nil {
		panic(err)
	}

	for appName, appHost := range routesMap {
		u := url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%s", eurekaHost, eurekaPort),
			Path:   fmt.Sprintf("/eureka/v2/apps/%s", appName),
		}

		// Resolve the IPv4 address of the host for registration with eureka:
		ip, err := net.ResolveIPAddr("ip4", appHost)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to resolve IPv4 for host '%s': %v\n", appName, appHost, err)
			continue
		}
		ipAddr := ip.String()

		port := 80
		hostPort := fmt.Sprintf("%s:%d", appHost, port)
		h := fnv.New64a()
		h.Write([]byte(hostPort))
		instanceHash := hex.EncodeToString(h.Sum(nil))
		instanceId := fmt.Sprintf("%s:%s", instanceHash, appName)

		register := map[string]interface{}{
			"instance": map[string]interface{}{
				"id":         appName,
				"instanceId": instanceId,
				"app":        appName,
				"hostName":   appHost,
				"ipAddr":     ipAddr,
				"port": map[string]interface{}{
					"@enabled": true,
					"$":        strconv.Itoa(port),
				},
				"securePort": map[string]interface{}{
					"@enabled": false,
					"$":        "443",
				},
				"dataCenterInfo": map[string]interface{}{
					"@class": "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
					"name":   "MyOwn",
				},
				"metadata": map[string]interface{}{
					"@class": "java.util.Collections$EmptyMap",
				},
				"vipAddress":       appHost,
				"secureVipAddress": appHost,
				"homePageUrl":      fmt.Sprintf("http://%s/", appHost),
				"statusPageUrl":    fmt.Sprintf("http://%s/info", appHost),
				"healthCheckUrl":   fmt.Sprintf("http://%s/health", appHost),
				// "secureHealthCheckUrl": fmt.Sprintf("http://%s/health", appHost),
			},
		}
		b, err := json.Marshal(&register)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
			continue
		}
		// body := bytes.NewReader(b)
		// io.Copy(os.Stdout, body)

		body := bytes.NewReader(b)

		// POST a body:
		rsp, err := http.DefaultClient.Post(u.String(), "application/json", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
			continue
		}

		if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
			fmt.Fprintf(os.Stderr, "%s: %v\n", appName, rsp.Status)
			io.Copy(os.Stderr, rsp.Body)
			fmt.Fprintln(os.Stderr)
			continue
		}

		fmt.Printf("Registered %s at '%s'\n", appName, appHost)
	}
}
