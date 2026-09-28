// Package npmregistry provides a minimal client for reading package metadata
// and tarball contents from the npm registry, scoped to what is needed to
// locate and extract renovate-schema.json for a given Renovate release.
package npmregistry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/maxcraig112/go-renovatebot/internal/versiontag"
)

const registryBaseURL = "https://registry.npmjs.org"

// Client talks to the npm registry over HTTP.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient returns a Client using http.DefaultClient.
func NewClient() *Client {
	return &Client{
		httpClient: http.DefaultClient,
		baseURL:    registryBaseURL,
	}
}

type packageMetadata struct {
	Versions map[string]versionMetadata `json:"versions"`
	DistTags map[string]string          `json:"dist-tags"`
}

type versionMetadata struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

// Versions returns every published stable release version of the given npm
// package, sorted ascending by semver. Prerelease versions (e.g.
// "40.0.0-next.1") are excluded.
func (c *Client) Versions(ctx context.Context, packageName string) ([]string, error) {
	meta, err := c.fetchMetadata(ctx, packageName)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(meta.Versions))
	for v := range meta.Versions {
		if !versiontag.IsValid(v) || versiontag.IsPrerelease(v) {
			continue
		}
		versions = append(versions, v)
	}
	versiontag.SortAscending(versions)
	return versions, nil
}

// LatestVersion returns the version npm currently tags as "latest".
func (c *Client) LatestVersion(ctx context.Context, packageName string) (string, error) {
	meta, err := c.fetchMetadata(ctx, packageName)
	if err != nil {
		return "", err
	}
	latest, ok := meta.DistTags["latest"]
	if !ok {
		return "", fmt.Errorf("npmregistry: no latest dist-tag for %s", packageName)
	}
	return latest, nil
}

func (c *Client) fetchMetadata(ctx context.Context, packageName string) (*packageMetadata, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, packageName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("npmregistry: fetching metadata for %s: %w", packageName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npmregistry: unexpected status %d fetching metadata for %s", resp.StatusCode, packageName)
	}

	var meta packageMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("npmregistry: decoding metadata for %s: %w", packageName, err)
	}
	return &meta, nil
}

// FetchTarballFile downloads the tarball for packageName@version and returns
// the contents of the given path within it (relative to the tarball root,
// e.g. "package/renovate-schema.json").
func (c *Client) FetchTarballFile(ctx context.Context, packageName, version, path string) ([]byte, error) {
	meta, err := c.fetchMetadata(ctx, packageName)
	if err != nil {
		return nil, err
	}

	versionMeta, ok := meta.Versions[version]
	if !ok {
		return nil, fmt.Errorf("npmregistry: version %s not found for %s", version, packageName)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionMeta.Dist.Tarball, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("npmregistry: downloading tarball for %s@%s: %w", packageName, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npmregistry: unexpected status %d downloading tarball for %s@%s", resp.StatusCode, packageName, version)
	}

	return extractFile(resp.Body, path)
}

func extractFile(r io.Reader, path string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("npmregistry: reading gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil, ErrFileNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("npmregistry: reading tar entry: %w", err)
		}
		if header.Name != path {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("npmregistry: reading %s: %w", path, err)
		}
		return data, nil
	}
}
