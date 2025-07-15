package deps

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/package-url/packageurl-go"
)

type Version struct {
	Version     string `bson:"version" json:"version"`
	PublishedAt string `bson:"publishedAt" json:"publishedAt"`
}

func (v *Version) Time() (time.Time, error) {
	layout := "2021-03-14T15:51:35Z"
	return time.Parse(layout, v.PublishedAt)

}

type DepsApiResponse struct {
	Versions []VersionsApiResponse `json:"versions"`
}

type VersionsApiResponse struct {
	Version     Version `json:"versionKey"`
	PublishedAt string  `bson:"publishedAt" json:"publishedAt"`
}

func GetVersions(rawPurl string) (*DepsApiResponse, error) {

	purl, err := packageurl.FromString(rawPurl)

	if err != nil {
		slog.Default().Warn("Purl parsing failed", "raw purl", purl)
		return nil, err
	}

	slog.Default().Debug("Deps.dev api query started", "purl", purl)

	system, name := getNameAndSystem(purl)
	if system == "" || name == "" {
		return nil, fmt.Errorf("Get name and system returned empty %+v", purl)
	}

	// GET /v3/systems/{packageKey.system}/packages/{packageKey.name}
	url := fmt.Sprintf("https://api.deps.dev/v3/systems/%s/packages/%s", system, name)
	slog.Default().Debug("Query constructed", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		slog.Default().Debug("Request failed with", "url", url, "err", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			retry := resp.Header.Get("Retry-After")
			slog.Default().Debug("Failed due to too many requests.", "retry-after", retry)
		}
		err := fmt.Errorf("request failed with status code %d", resp.StatusCode)
		slog.Default().Debug("Request failed with", "url", url, "err", err.Error())
		return nil, err
	}

	var deps DepsApiResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&deps); err != nil {
		slog.Default().Debug("Decoding of response failed", "url", url, "err", err.Error())
		return nil, err
	}

	return &deps, nil
}

func getNameAndSystem(purl packageurl.PackageURL) (name, system string) {
	namespace := purl.Namespace
	if namespace != "" {
		namespace = namespace + "/"
	}

	// deps.dev sytem:
	// Can be one of GO, RUBYGEMS, NPM, CARGO, MAVEN, PYPI, NUGET.
	name = namespace + purl.Name
	switch purl.Type {
	case packageurl.TypeNPM:
		system = "npm"
	case packageurl.TypeGolang:
		system = "go"
	case packageurl.TypeCargo:
		system = "cargo"
	case packageurl.TypeMaven, packageurl.TypeGradle:
		system = "maven"
		name = purl.Namespace + ":" + purl.Name
	case packageurl.TypePyPi:
		system = "pypi"
	case packageurl.TypeNuget:
		system = "nuget"
	case packageurl.TypeGem:
		system = "rubygems"
	}

	name = url.PathEscape(name)
	system = url.PathEscape(system)

	return name, system
}
