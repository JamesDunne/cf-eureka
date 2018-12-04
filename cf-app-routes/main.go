package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
)

// Execute a CloudFoundry authenticated API request via `cf curl` and return
// the JSON result in `result`:
func execCF(route string, result interface{}) error {
	// Execute local `cf` curl command:
	cmd := exec.Command("cf", "curl", route)

	// TODO: capture stderr too?
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
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Expected <org name> <space name> arguments to query routes for")
		os.Exit(1)
		return
	}

	// Create JSON encoder to write to stdout:
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// Get org name from first argument:
	orgName := args[0]
	fmt.Fprintf(os.Stderr, "Query cf for ID of org '%s'\n", orgName)

	// Look for orgs with the name specified:
	var orgResults map[string]interface{}
	execCF("/v2/organizations?q=name:"+url.QueryEscape(orgName), &orgResults)

	orgTotalResults := orgResults["total_results"].(float64)
	if orgTotalResults == 0 {
		fmt.Fprintf(os.Stderr, "No orgs found matching name='%s'\n", orgName)
		os.Exit(2)
		return
	}
	if orgTotalResults > 1 {
		fmt.Fprintf(os.Stderr, "Too many orgs found matching name='%s'\n", orgName)
		os.Exit(2)
		return
	}

	orgMetadata := orgResults["resources"].([]interface{})[0].(map[string]interface{})["metadata"].(map[string]interface{})
	orgGuid := orgMetadata["guid"].(string)

	// This is "nu"
	//orgGuid := "3c5840c6-748b-4da2-8e2b-9d2e685993f1"

	fmt.Fprintf(os.Stderr, "Org '%s' has ID '%s'\n", orgName, orgGuid)

	// Get space name from second argument:
	spaceName := args[1]
	fmt.Fprintf(os.Stderr, "Query cf for ID of space '%s'\n", spaceName)

	// Look for spaces with the name specified within the selected org:
	var spaceResults map[string]interface{}
	execCF("/v2/organizations/"+url.QueryEscape(orgGuid)+"/spaces?q=name:"+url.QueryEscape(spaceName), &spaceResults)

	spaceTotalResults := spaceResults["total_results"].(float64)
	if spaceTotalResults == 0 {
		fmt.Fprintf(os.Stderr, "No spaces found matching name='%s'\n", spaceName)
		os.Exit(2)
		return
	}
	if spaceTotalResults > 1 {
		fmt.Fprintf(os.Stderr, "Too many spaces found matching name='%s'\n", spaceName)
		os.Exit(2)
		return
	}

	spaceMetadata := spaceResults["resources"].([]interface{})[0].(map[string]interface{})["metadata"].(map[string]interface{})
	spaceGuid := spaceMetadata["guid"].(string)

	// This is "dev" for "nu"
	//spaceGuid := "97512eba-2f24-43ce-86f4-f96d3a459ed0"

	fmt.Fprintf(os.Stderr, "Space '%s' has ID '%s'\n", spaceName, spaceGuid)

	appMap := make(map[string]string)

	// Get all the NU space's routes:
	nextURL := fmt.Sprintf(
		"/v2/spaces/%s/routes?results-per-page=100&page=1&inline-relations-depth=1",
		spaceGuid,
	)

	pageNumber := 1
	for nextURL != "" {
		var results map[string]interface{}
		fmt.Fprintf(os.Stderr, "Fetching page %d of routes for space...\n", pageNumber)
		if err := execCF(nextURL, &results); err != nil {
			fmt.Fprintf(os.Stderr, "Error encountered: %v\n", err)
			return
		}
		//fmt.Fprintf(os.Stderr, "Fetched page %d\n", pageNumber)

		resources := results["resources"].([]interface{})
		for _, r := range resources {
			//enc.Encode(r)
			//fmt.Print("\n")

			rs := r.(map[string]interface{})

			// entity.apps[0].entity.name
			//       .host
			//       .port
			//       .path
			// entity.domain.entity.name

			entity := rs["entity"].(map[string]interface{})
			//json.NewEncoder(os.Stderr).Encode(entity)

			domainEntity := entity["domain"].(map[string]interface{})["entity"].(map[string]interface{})

			apps := entity["apps"].([]interface{})
			if apps == nil || len(apps) == 0 {
				continue
			}

			appEntity := apps[0].(map[string]interface{})["entity"].(map[string]interface{})

			appName := appEntity["name"].(string)
			appHost := entity["host"].(string)
			domainName := domainEntity["name"].(string)

			fmt.Fprintf(os.Stderr, "    \"%s\": \"%s.%s\"\n", appName, appHost, domainName)

			// TODO: include port and path maybe?
			appMap[appName] = fmt.Sprintf("%s.%s", appHost, domainName)
		}

		// Find next URL for paging:
		if results["next_url"] == nil {
			break
		}
		nextURL = results["next_url"].(string)
		// TODO: we could always just parse nextURL and rip out the `page` query-string parameter.
		pageNumber++
	}

	// Send final output as JSON to stdout:
	fmt.Fprintf(os.Stderr, "Encoding final output as JSON\n")
	if err := enc.Encode(appMap); err != nil {
		panic(err)
	}
}
