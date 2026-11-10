// configtool shares dotenv parsing and validation with AppView. It never builds dependencies.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"social.craftsky/appview/internal/app"
)

func main() {
	if len(os.Args) < 3 {
		fail()
	}
	switch os.Args[1] {
	case "marshal":
		values := map[string]string{}
		if err := json.NewDecoder(os.Stdin).Decode(&values); err != nil {
			fail()
		}
		// Always quote strings: godotenv.Marshal coerces numeric-looking values,
		// which would corrupt credentials and identifiers with leading zeroes.
		lines := []string{}
		escape := strings.NewReplacer("\\", "\\\\", "\n", `\n`, "\r", `\r`, `"`, `\"`, "$", `\$`)
		for key, value := range values {
			lines = append(lines, key+`="`+escape.Replace(value)+`"`)
		}
		sort.Strings(lines)
		_ = json.NewEncoder(os.Stdout).Encode(strings.Join(lines, "\n"))
	case "read":
		values, err := godotenv.Read(os.Args[2])
		if err != nil {
			fail()
		}
		_ = json.NewEncoder(os.Stdout).Encode(values)
	case "validate":
		root := os.Args[2]
		// Files are applied in the same order as Compose; inherited environment wins.
		values := map[string]string{}
		for _, name := range []string{"appview/environments/dev.env", ".env.secrets.local", ".env.local"} {
			path := filepath.Join(root, name)
			next, err := godotenv.Read(path)
			if os.IsNotExist(err) && name != "appview/environments/dev.env" {
				continue
			}
			if err != nil {
				fail()
			}
			for k, v := range next {
				values[k] = v
			}
		}
		// Match the container-only topology overrides. No network access is made.
		values["DATABASE_URL"] = "postgres://craftsky:dev@postgres:5432/craftsky_dev?sslmode=disable"
		values["TAP_WS_URL"] = "ws://tap:2480/channel"
		values["APPVIEW_ENV"] = "dev"
		host := os.Getenv("CRAFTSKY_APPVIEW_PUBLISH_HOST")
		if host == "" {
			host = values["CRAFTSKY_APPVIEW_PUBLISH_HOST"]
		}
		if host == "" {
			host = "127.0.0.1"
		}
		values["APPVIEW_PUBLISH_HOST"] = host
		values["SCHEDULED_POSTS_S3_ENDPOINT"] = "http://minio:9000"
		values["SAFETY_EVIDENCE_S3_ENDPOINT"] = "http://minio:9000"
		for k, v := range values {
			if _, ok := os.LookupEnv(k); !ok {
				_ = os.Setenv(k, v)
			}
		}
		_, err := app.LoadConfig(app.EnvDev, filepath.Join(root, ".nonexistent-config"))
		if err != nil {
			// Never print parser errors: they can contain protected input values.
			keys := []string{}
			if strings.Contains(err.Error(), "bunny") {
				keys = append(keys, "BUNNY_STREAM_LIBRARY_ID", "BUNNY_STREAM_API_KEY", "BUNNY_STREAM_CDN_HOSTNAME")
			}
			reference, _ := os.ReadFile(filepath.Join(root, "docs/development/configuration-reference.md"))
			for _, match := range regexp.MustCompile("\\| `([A-Z][A-Z0-9_]+)`").FindAllStringSubmatch(string(reference), -1) {
				if strings.Contains(err.Error(), match[1]) {
					keys = append(keys, match[1])
				}
			}
			sort.Strings(keys)
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": false, "keys": keys})
			os.Exit(1)
		}
		fmt.Println(`{"valid":true}`)
	default:
		fail()
	}
}
func fail() {
	fmt.Fprintln(os.Stderr, "configtool: unable to read or parse configuration (values withheld)")
	os.Exit(1)
}
