package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Execute a CloudFoundry authenticated API request via `cf curl` and return
// the JSON result in `result`:
func execCF(route string, result interface{}) error {
	// Execute local `cf` curl command:
	cmd := exec.Command("cf", "curl", route)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer stdout.Close()

	// JSON decode the stdout:
	if err = json.NewDecoder(stdout).Decode(result); err != nil {
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func main() {
	var results map[string]interface{}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// Get all the NU space's routes:
	nextURL := "/v2/spaces/97512eba-2f24-43ce-86f4-f96d3a459ed0/routes?results-per-page=100&page=1&inline-relations-depth=1"
	for nextURL != "" {
		if err := execCF(nextURL, &results); err != nil {
			panic(err)
		}

		resources := results["resources"].([]interface{})
		for _, r := range resources {
			_ = enc
			//enc.Encode(r)
			//fmt.Print("\n")

			rs := r.(map[string]interface{})

			entity := rs["entity"].(map[string]interface{})
			domainEntity := entity["domain"].(map[string]interface{})["entity"].(map[string]interface{})

			apps := entity["apps"].([]interface{})
			if apps == nil || len(apps) == 0 {
				continue
			}

			appEntity := apps[0].(map[string]interface{})["entity"].(map[string]interface{})

			appName := appEntity["name"].(string)
			appHost := entity["host"].(string)
			domainName := domainEntity["name"].(string)

			fmt.Printf("\t%s: %s.%s\n", appName, appHost, domainName)
			// entity.apps[0].entity.name
			//       .host
			//       .port
			//       .path
			// entity.domain.entity.name
		}

		// Find next URL for paging:
		if results["next_url"] == nil {
			break
		}
		nextURL = results["next_url"].(string)
	}
}
