package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sphireinc/go-sdkgen/internal/generator"
)

func main() {
	var (
		in         = flag.String("input", "", "Path to swagger.json (OpenAPI v2)")
		out        = flag.String("out", "./sdk", "Output directory for generated SDK")
		lang       = flag.String("lang", "ts", "Language: ts or js")
		name       = flag.String("name", "GeneratedSDK", "SDK name used in banners/comments")
		baseUrlVar = flag.String("baseUrlVar", "baseApiUrl", "Exported baseUrl variable name")
		auth       = flag.String("auth", "bearer", "Auth mode: none|bearer")
		tokenFn    = flag.String("tokenFn", "getToken", "Token function name used in generated requests")
	)
	flag.Parse()
	if *in == "" {
		_, err := fmt.Fprintln(os.Stderr, "missing --input")
		if err != nil {
			return
		}
		os.Exit(2)
	}
	absOut, _ := filepath.Abs(*out)

	cfg := generator.Config{
		InputPath:  *in,
		OutputDir:  absOut,
		Lang:       *lang,
		SDKName:    *name,
		BaseURLVar: *baseUrlVar,
		AuthMode:   *auth,
		TokenFn:    *tokenFn,
	}

	if err := generator.Generate(cfg); err != nil {
		_, err := fmt.Fprintln(os.Stderr, "generate failed:", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	fmt.Println("SDK generated at:", absOut)
}
