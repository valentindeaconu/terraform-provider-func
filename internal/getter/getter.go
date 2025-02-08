package getter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-getter"
	urlhelper "github.com/hashicorp/go-getter/helper/url"
)

type FetchInput struct {
	// URL represents the url from which the file should be downloaded.
	URL string

	// Checksum represents the checksum of the file to be checked against.
	Checksum string

	// Path represents the path where the file should be stored after it
	// was downloaded.
	Path string
}

// Fetch downloads a file/directory from a given URL.
//
// It computes a hash of the source and then generates a key for the file.
// If that exact key already exists in the destination path, the entire
// download process is skipped.
//
// The method is not checking the file content, only its source and name.
func Fetch(ctx context.Context, in *FetchInput) (string, error) {
	u, err := urlhelper.Parse(in.URL)
	if err != nil {
		return "", err
	}

	// Set extra arguments
	q := u.Query()
	q.Add("archive", "false")

	if in.Checksum != "" {
		q.Add("checksum", in.Checksum)
	}

	u.RawQuery = q.Encode()

	filename := filepath.Base(u.Path)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	ext = strings.TrimPrefix(ext, ".")

	// Compute the hash of the URL
	h := sha1.New()
	h.Write([]byte(in.URL))
	hash := hex.EncodeToString(h.Sum(nil))

	key := fmt.Sprintf("%s.%s.%s", name, hash, ext)
	dst := filepath.Join(in.Path, key)

	if _, err := os.Stat(dst); err == nil {
		// This exact file was already downloaded. We can skip the download.
		return dst, nil
	}

	// Configure the client
	ctx, cancel := context.WithCancel(ctx)
	client := &getter.Client{
		Ctx:  ctx,
		Src:  u.String(),
		Dst:  dst,
		Pwd:  dst,
		Mode: getter.ClientModeFile,
	}

	// Launch the download process
	wg := sync.WaitGroup{}
	wg.Add(1)
	ech := make(chan error, 2)
	go func() {
		defer wg.Done()
		defer cancel()

		if err := client.Get(); err != nil {
			ech <- err
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)

	// Wait for the download process to finish
	select {
	case sig := <-sc:
		signal.Reset(os.Interrupt)
		cancel()
		wg.Wait()

		return "", fmt.Errorf("download canceled: signal %v received", sig.String())
	case err := <-ech:
		wg.Wait()

		return "", fmt.Errorf("could not download resource: %v", err)
	case <-ctx.Done():
		wg.Wait()

		return dst, nil
	}
}
