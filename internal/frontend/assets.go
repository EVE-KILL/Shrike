package frontend

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	dogmaassets "github.com/eve-kill/shrike/frontend/dogma"
)

//go:embed static
var assets embed.FS

type asset struct {
	data        []byte
	etag        string
	contentType string
}

type assetManifest struct {
	revision string
	files    map[string]asset
}

// One revision covers the complete asset tree. CSS and JS references get the
// same revision, so a cached stylesheet never points at a different build.
var manifest = buildAssets()

// Match the start of a URL in CSS/JS, rather than a path segment inside an
// API URL such as /api/user/domains/1/assets/4 or a third-party URL.
var localAssetReference = regexp.MustCompile("([\"'`(]|^)/assets/")

func versionAssetReferences(data []byte, revision string) []byte {
	return localAssetReference.ReplaceAll(data, []byte("${1}/assets/"+revision+"/"))
}

func buildAssets() assetManifest {
	files := map[string]asset{}
	hash := sha256.New()
	// The immutable URL also identifies the transformation applied to source
	// bytes. Bump this when changing URL rewriting, even if files are unchanged.
	_, _ = hash.Write([]byte("root-asset-url-prefix-v2\x00"))
	err := fs.WalkDir(assets, "static", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		name = strings.TrimPrefix(name, "static/")
		// WalkDir is lexical; names and lengths avoid ambiguous concatenations.
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", name, len(data))
		_, _ = hash.Write(data)
		files[name] = asset{data: data}
		return nil
	})
	if err != nil {
		panic(err)
	}
	// Keep configurable Lucide symbols separate from the small shared sprite.
	// They use the normal immutable asset transport; no frontend runtime or
	// icon API is required on a cache miss.
	var icons map[string]struct {
		ViewBox string
		Body    string
	}
	if file, ok := files["lucide-icons.json"]; ok {
		if err := json.Unmarshal(file.data, &icons); err != nil {
			panic(err)
		}
		for name, icon := range icons {
			data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><symbol id="icon" viewBox="` + icon.ViewBox + `">` + icon.Body + `</symbol></svg>`)
			files["lucide/"+name+".svg"] = asset{data: data}
		}
		delete(files, "lucide-icons.json")
	}
	// Dogma's source data is versioned with the shared Rust adapter. Embed it
	// once and publish it through the same immutable asset manifest.
	err = fs.WalkDir(dogmaassets.Data, "dist/upstream", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := dogmaassets.Data.ReadFile(name)
		if err != nil {
			return err
		}
		assetName := "dogma/" + path.Base(name)
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", assetName, len(data))
		_, _ = hash.Write(data)
		files[assetName] = asset{data: data}
		return nil
	})
	if err != nil {
		panic(err)
	}
	result := assetManifest{revision: fmt.Sprintf("%x", hash.Sum(nil))[:16], files: files}
	for name, file := range files {
		switch path.Ext(name) {
		case ".css", ".js":
			file.data = versionAssetReferences(file.data, result.revision)
		}
		file.etag = fmt.Sprintf(`"%x"`, sha256.Sum256(file.data))
		file.contentType = mime.TypeByExtension(path.Ext(name))
		if file.contentType == "" {
			file.contentType = http.DetectContentType(file.data)
		}
		files[name] = file
	}
	return result
}

func assetURL(name string) string {
	name = strings.TrimPrefix(name, "/assets/")
	return "/assets/" + manifest.revision + "/" + name
}

func serveAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")
	versioned := strings.HasPrefix(name, manifest.revision+"/")
	if versioned {
		name = strings.TrimPrefix(name, manifest.revision+"/")
	}
	file, exists := manifest.files[name]
	if !exists || !fs.ValidPath(name) {
		w.Header().Set("Cache-Control", "no-store")
		http.NotFound(w, r)
		return
	}
	cache := "public, max-age=0, must-revalidate"
	if versioned {
		cache = "public, max-age=31536000, immutable"
	}
	w.Header().Set("Cache-Control", cache)
	w.Header().Set("ETag", file.etag)
	w.Header().Set("Content-Type", file.contentType)
	// ServeContent implements conditional requests, HEAD and byte ranges.
	http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(file.data))
}
