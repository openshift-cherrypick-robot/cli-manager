package image

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/openshift/cli-manager/api/v1alpha1"
)

const TarballPath = "/var/run/plugins/"

// Pull an image down to the local filesystem.
func Pull(src string, auth string, platform *v1.Platform, ca string, proxy *url.URL) (v1.Image, error) {
	craneOptions := []crane.Option{}
	if len(auth) > 0 {
		auth := authn.FromConfig(authn.AuthConfig{
			Auth: auth,
		})
		craneOptions = append(craneOptions, crane.WithAuth(auth))
	}

	if platform != nil {
		craneOptions = append(craneOptions, crane.WithPlatform(platform))
	}

	transport := remote.DefaultTransport.(*http.Transport).Clone()
	if ca != "" {
		caBytes, err := base64.StdEncoding.DecodeString(ca)
		if err != nil {
			return nil, fmt.Errorf("error decoding CA certificate: %w", err)
		}
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("invalid CA certificate passed in")
		}
		if !rootCAs.Equal(x509.NewCertPool()) {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}
			transport.TLSClientConfig.RootCAs = rootCAs
		}
	}

	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}

	var rt http.RoundTripper = transport
	craneOptions = append(craneOptions, crane.WithTransport(rt))
	return crane.Pull(src, craneOptions...)
}

// Extract an image's filesystem as a tarball, or individual files from the image.
func Extract(img v1.Image, platform v1alpha1.PluginPlatform, destinationName string) ([]v1alpha1.FileLocation, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("retrieving image layers: %v", err)
	}

	processedTargets := make(map[string]struct{})

	file, err := os.Create(destinationName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	foundLen := 0
	// we iterate through the layers in reverse order because it makes handling
	// whiteout layers more efficient, since we can just keep track of the removed
	// files as we see .wh. layers and ignore those in previous layers.
	for i := len(layers) - 1; i >= 0; i-- {
		if foundLen == len(platform.Files) {
			break
		}
		layer := layers[i]
		layerReader, err := layer.Uncompressed()
		if err != nil {
			return nil, fmt.Errorf("reading layer contents: %v", err)
		}

		tarReader := tar.NewReader(layerReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				layerReader.Close()
				return nil, fmt.Errorf("reading tar: %v", err)
			}

			// skip directories
			if header.Typeflag == tar.TypeDir {
				continue
			}

			// skip empty file contents
			if header.Size == 0 {
				continue
			}

			// some tools prepend everything with "./", so if we don't Clean the
			// name, we may have duplicate entries, which angers tar-split.
			header.Name = filepath.Clean(header.Name)

			// skip empty file names
			if len(header.Name) == 0 {
				continue
			}

			// skip the file if it was already found and processed in a previous/more recent layer
			if _, ok := processedTargets[header.Name]; ok {
				continue
			}

			// determine if we care about the given file
			for _, target := range platform.Files {
				if header.Name == strings.TrimPrefix(target.From, "/") {
					processedTargets[target.From] = struct{}{}
					// TODO: Should we write it to target.To?
					if err := tw.WriteHeader(header); err != nil {
						continue
					}

					if _, err := io.Copy(tw, tarReader); err != nil {
						continue
					}
					foundLen++
					break
				}
			}
			if foundLen == len(platform.Files) {
				break
			}
		}
		layerReader.Close()
	}

	var fileLocation []v1alpha1.FileLocation
	for _, f := range platform.Files {
		if _, ok := processedTargets[f.From]; ok {
			fileLocation = append(fileLocation, f)
		}
	}

	return fileLocation, nil
}
